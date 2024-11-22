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
		// 翻译标题
		if article.Title != "" && article.TitleCN == "" {
			if article.From == "jinse" || article.From == "bitpie" { // 金色财经和比特派为中文数据
				article.TitleCN = article.Title
				article.Title, _ = translator.Send(article.Title)
				article.Title = strings.Split(article.Title, "\n")[0]
			} else {
				article.TitleCN, _ = translator.Send(article.Title)
				article.TitleCN = strings.Split(article.TitleCN, "\n")[0]
			}
		}

		// 翻译简介
		if article.Abstract != "" && article.AbstractCN == "" && !strings.HasSuffix(article.From, "_coin") {
			if article.From == "jinse" || article.From == "bitpie" {
				article.AbstractCN = article.Abstract
				article.Abstract, _ = translator.Send(article.Abstract)
			} else {
				article.AbstractCN, _ = translator.Send(article.Abstract)
			}
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
func newQueueWrapper(ctx context.Context, q *queue.Queue) newsaddr.QueueWrapper {
	var (
		count = 10
		ch    = make(chan models.Article, 20)
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

		var (
			list   = make([]models.Article, 0, 100)
			isOver = false
		)
		for {
			select {
			case article := <-ch:
				list = append(list, article)
			case <-ctx.Done():
				isOver = true
			default:
				if len(list) > 0 {
					index := count
					if len(list) < count {
						index = len(list)
					}

					items := list[0:index]
					list = list[index:]

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
				} else if isOver {
					logger.Infof("All articles have been sent to the queue. Quitting...")
					return
				}
			}
		}
	}()

	return func(articles ...models.Article) {
		for i := range articles {
			article := articles[i]
			if article.Title == "" {
				continue
			}

			if article.Image == "" {
				logger.Infof("starting search for image %s", article.Title)
				try := 5

			SEARCH:
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				g := utils.NewGoogleSearch(ctx)
				article.Image, _ = g.Search(article.Title)
				g.Close()
				cancel()
				if article.Image == "" {
					if try <= 0 {
						logger.Infof("Failed to find image for %s", article.Title)
						continue
					}
					logger.Infof("Retrying search for image %s", article.Title)
					time.Sleep(3 * time.Second)
					try--
					goto SEARCH
				}

				logger.Infof("search result: %s", article.Image)
			}
			ch <- article
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
				return nil
			}
		}

		// 保存文章信息
		if strings.HasSuffix(article.From, "_coin") { // 新闻来源是币
			if err := s.SaveCoin(article); err != nil {
				return err
			}
		} else {
			if err := s.Save(article); err != nil {
				return err
			}
		}
		logger.Infof("Saved article: %s, link: %s", article.Title, article.Link)

		return nil
	}))
}

func getScrapers(ctx context.Context) ([]newsaddr.Scraper, *queue.Queue) {
	threshold := config.Cfg.Scrapy.Threshold
	q := newQueue(
		translateTitle(),
		removeDuplicates(threshold),
	)

	qw := newQueueWrapper(ctx, q)
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
		newsaddr.NewBaiduScrapy(qw),
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scrapers, q := getScrapers(ctx)
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

	logger.Infof("Queue task finished. submitted tasks: %d, success tasks: %d, failure tasks: %d", q.SubmittedTasks(), q.SuccessTasks(), q.FailureTasks())
	logger.Info("Task started successfully.")
}
