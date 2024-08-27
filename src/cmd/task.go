package cmd

import (
	"context"
	"encoding/json"
	"github.com/golang-queue/queue"
	"github.com/golang-queue/queue/core"
	"news/src/logger"
	"news/src/models"
	"news/src/newsaddr"
	"news/src/storage"
)

var (
	scrapers []newsaddr.Scraper
	q        *queue.Queue
)

func newQueue() *queue.Queue {
	s := storage.NewService()
	return queue.NewPool(10, queue.WithFn(func(ctx context.Context, m core.QueuedMessage) error {
		article := &models.Article{}
		if err := json.Unmarshal(m.Bytes(), article); err != nil {
			logger.Errorf("Failed to unmarshal message: %s", err)
			return err
		}

		logger.Infof("Scraped article[%s]: %s, link: %s", article.Token, article.Title, article.Link)
		if err := s.Save(article); err != nil {
			logger.Errorf("Failed to save article: %s", err)
			return err
		}

		return nil
	}))
}

func StartTask() {
	logger.Info("Starting task...")

	// start scraping
	defer q.Release()
	for _, c := range scrapers {
		if err := c.Run(); err != nil {
			logger.Errorf("Task failed: %s", err)
			return
		}
	}

	logger.Info("Task started successfully.")
}

func init() {
	q = newQueue()
	scrapers = []newsaddr.Scraper{
		newsaddr.NewJinSeScrapy(q),
		newsaddr.NewBeinCryptoScrapy(q),
		newsaddr.NewBlockWorksScrapy(q),
		newsaddr.NewCoinDeskScrapy(q),
		newsaddr.NewTheBlockScrapy(q),
		newsaddr.NewDecryptScrapy(q),
		newsaddr.NewTheDefiantScrapy(q),
	}
}
