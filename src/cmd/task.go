package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/golang-queue/queue"
	"github.com/golang-queue/queue/core"
	"news/src/config"
	"news/src/logger"
	"news/src/models"
	"news/src/newsaddr"
	"news/src/storage"
	"news/src/utils"
	"reflect"
	"strings"
	"sync"
	"time"
)

var s *storage.Service

type pluginFunc func(article *models.Article) error

func translateTitle() pluginFunc {
	translator, err := utils.NewTranslate(config.Cfg.Kimi.Prompt)
	if err != nil {
		panic(err)
	}

	return func(article *models.Article) error {
		if article.TitleCN != "" {
			return nil
		}

		if article.From == "jinse" || article.From == "bitpie" { // 金色财经和比特派为中文数据
			article.TitleCN = article.Title
			article.Title, _ = translator.Send(article.Title)
			article.Title = strings.Split(article.Title, "\n")[0]
		} else if article.Title != "" {
			article.TitleCN, _ = translator.Send(article.Title)
			article.TitleCN = strings.Split(article.TitleCN, "\n")[0]
		}

		return nil
	}
}

func searchImage(ctx context.Context) pluginFunc {
	g := utils.NewGoogleSearch(ctx)
	return func(article *models.Article) error {
		if article.Image == "" {
			article.Image, _ = g.Search(article.Title)
		}

		return nil
	}
}

func removeDuplicates(threshold float64) pluginFunc {
	lock := sync.Mutex{}
	tm := make(map[string][]string)
	return func(article *models.Article) error {
		lock.Lock()
		defer lock.Unlock()

		key := string(article.Category)
		titles, ok := tm[key]
		if !ok {
			titles = make([]string, 0)
		}

		if article.Title != "" {
			if utils.IsUniqueStrings(titles, article.Title, threshold) {
				titles = append(titles, article.Title)
				tm[key] = titles
			} else {
				logger.Infof("Duplicate title: %s, link: %s", article.Title, article.Link)
				return errors.New("duplicate title")
			}
		}

		return nil
	}
}

// newQueueWrapper wraps a queue to send articles to it.
func newQueueWrapper(q *queue.Queue, quit <-chan struct{}) newsaddr.QueueWrapper {
	var (
		mu     sync.Mutex
		count  = 10
		list   = make([]models.Article, 0, 100)
		isOver = false
	)
	translator, err := utils.NewTranslate(config.Cfg.Kimi.Prompt)
	if err != nil {
		logger.Errorf("Failed to initialize translation: %s", err)
	}
	if config.Cfg.Kimi.Tokens > 0 {
		count = config.Cfg.Kimi.Tokens
	}

	go func() {
		defer func() {
			q.Release() // Release the queue when it's done.
			logger.Infof("Finished sending articles to queue.")
		}()

		for {
			mu.Lock()
			if len(list) > 0 {
				index := count
				if len(list) < count {
					index = len(list)
				}

				items := list[0:index]
				list = list[index:]
				mu.Unlock()

				// translate the articles
				titles := make([]string, len(items))
				for i, article := range items {
					titles[i] = article.Title
				}
				if r, err := translator.Send(titles...); err == nil {
					if strings.Contains(r, "\n\n") {
						titles = strings.Split(r, "\n\n")
					} else {
						titles = strings.Split(r, "\n")
					}
				}

				for i, article := range items {
					if i < len(titles) {
						if article.From == "jinse" || article.From == "bitpie" {
							article.TitleCN = article.Title
							article.Title = titles[i]
						} else {
							article.TitleCN = titles[i]
						}
					}
					if err = q.Queue(&article); err != nil {
						logger.Errorf("Failed to send article: %s", err)
					}
				}
				logger.Infof("Sent %d articles to the queue", len(items))
			} else {
				mu.Unlock()
			}

			select {
			case <-quit:
				isOver = true
			default:
				if isOver && len(list) == 0 {
					logger.Infof("All articles have been sent to the queue. Quitting...")
					return
				} else {
					time.Sleep(time.Second * 3)
				}
			}
		}
	}()

	return func(articles ...models.Article) {
		mu.Lock()
		defer mu.Unlock()

		for i := range articles {
			article := articles[i]
			if article.Title == "" {
				continue
			}

			if article.Image == "" {
				logger.Infof("starting search for image %s", article.Title)

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				g := utils.NewGoogleSearch(ctx)
				article.Image, _ = g.Search(article.Title)
				g.Close()
				cancel()

				logger.Infof("search result: %s", article.Image)
			}
			list = append(list, article)
		}
	}
}

func newQueue(plugins ...pluginFunc) *queue.Queue {
	return queue.NewPool(5, queue.WithLogger(logger.GetLogger()), queue.WithFn(func(ctx context.Context, m core.QueuedMessage) error {
		article := &models.Article{}
		if err := json.Unmarshal(m.Bytes(), article); err != nil {
			logger.Errorf("Failed to unmarshal message: %s", err)
			return err
		}

		logger.Infof("Scraped article: %s, link: %s", article.Title, article.Link)

		// 检查文章信息，自动填充缺失信息
		for _, p := range plugins {
			if err := p(article); err != nil {
				logger.Errorf("Failed to process article: %s", err)
				return nil
			}
		}

		// 保存文章信息
		if strings.HasSuffix(article.From, "_coin") { // 新闻来源是币
			if err := s.SaveCoin(article); err != nil {
				logger.Errorf("Failed to save coin article: %s", err)
				return err
			}
		} else {
			if err := s.Save(article); err != nil {
				logger.Errorf("Failed to save article: %s", err)
				return err
			}
		}
		logger.Infof("Saved article: %s, link: %s", article.Title, article.Link)

		return nil
	}))
}

func getScrapers(quit <-chan struct{}) ([]newsaddr.Scraper, *queue.Queue) {
	threshold := config.Cfg.Scrapy.Threshold
	q := newQueue(
		translateTitle(),
		removeDuplicates(threshold),
	)

	qw := newQueueWrapper(q, quit)
	scrapers := []newsaddr.Scraper{
		newsaddr.NewJinSeScrapy(qw),
		newsaddr.NewBeinCryptoScrapy(qw),
		newsaddr.NewBlockWorksScrapy(qw),
		newsaddr.NewCoinDeskScrapy(qw),
		newsaddr.NewTheBlockScrapy(qw),
		newsaddr.NewDecryptScrapy(qw),
		newsaddr.NewTheDefiantScrapy(qw),
		newsaddr.NewBinanceScrapy(qw),
		newsaddr.NewBitPieScrapy(qw),
	}

	return scrapers, q
}

func StartScrapyTask() {
	logger.Info("Starting task...")

	// restore articles from storage if exists
	dataVersion := time.Now().Unix()
	s = storage.NewServiceWithVersion(dataVersion)
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("Scrapy task failed: %s", err)
			return
		}

		s.SetVersion(dataVersion)
		storage.NotifyVersion(dataVersion)
		if err := s.Restore(); err != nil {
			logger.Errorf("Restoring articles failed: %s", err)
			return
		}
	}()

	// start scraping
	quit := make(chan struct{})
	scrapers, q := getScrapers(quit)
	for _, c := range scrapers {
		name := reflect.TypeOf(c).Elem().Name()
		logger.Infof("[%s]Startup scrapy...", name)

		start := time.Now()
		if err := c.Run(); err != nil {
			logger.Errorf("Task failed: %s", err)
			return
		}
		elapsed := time.Since(start)

		logger.Infof("[%s]Finished scrapy. elapsed time: %s", name, elapsed)
	}

	// wait for all queue tasks to finish
	for q.BusyWorkers() > 0 || q.SuccessTasks()+q.FailureTasks() < q.SubmittedTasks() {
		logger.Infof("Waiting for queue tasks to finish. busy workers: %d", q.BusyWorkers())
		time.Sleep(time.Second)
	}
	quit <- struct{}{}

	logger.Infof("Queue task finished. submitted tasks: %d, success tasks: %d, failure tasks: %d", q.SubmittedTasks(), q.SuccessTasks(), q.FailureTasks())
	logger.Info("Task started successfully.")
}
