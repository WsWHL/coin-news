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

var (
	scrapers []newsaddr.Scraper
	q        *queue.Queue
	s        *storage.Service
)

type pluginFunc func(article *models.Article) error

func translateTitle() pluginFunc {
	translator, err := utils.NewTranslate(config.Cfg.Kimi.Prompt)
	if err != nil {
		panic(err)
	}

	return func(article *models.Article) error {
		if article.From == "jinse" || article.From == "bitpie" { // 金色财经和比特派为中文数据
			article.TitleCN = article.Title
			article.Title, _ = translator.Send(article.Title)
		} else if article.Title != "" && article.TitleCN == "" {
			article.TitleCN, _ = translator.Send(article.Title)
		}

		return nil
	}
}

func searchImage() pluginFunc {
	g := utils.NewGoogleSearch(context.Background())
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
	defer q.Release()
	for q.BusyWorkers() > 0 {
		logger.Infof("Waiting for queue tasks to finish. busy workers: %d", q.BusyWorkers())
		time.Sleep(time.Second)
	}

	logger.Infof("Queue task finished. submitted tasks: %d, success tasks: %d, failure tasks: %d", q.SubmittedTasks(), q.SuccessTasks(), q.FailureTasks())
	logger.Info("Task started successfully.")
}

func init() {
	threshold := config.Cfg.Scrapy.Threshold
	q = newQueue(
		translateTitle(),
		removeDuplicates(threshold),
		searchImage(),
	)

	scrapers = []newsaddr.Scraper{
		newsaddr.NewJinSeScrapy(q),
		newsaddr.NewBeinCryptoScrapy(q),
		newsaddr.NewBlockWorksScrapy(q),
		newsaddr.NewCoinDeskScrapy(q),
		newsaddr.NewTheBlockScrapy(q),
		newsaddr.NewDecryptScrapy(q),
		newsaddr.NewTheDefiantScrapy(q),
		newsaddr.NewBinanceScrapy(q),
		newsaddr.NewBitPieScrapy(q),
	}
}
