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
	"strings"
)

var (
	scrapers []newsaddr.Scraper
	q        *queue.Queue
	s        *storage.Service
)

func newQueue() *queue.Queue {
	return queue.NewPool(5, queue.WithFn(func(ctx context.Context, m core.QueuedMessage) error {
		article := &models.Article{}
		if err := json.Unmarshal(m.Bytes(), article); err != nil {
			logger.Errorf("Failed to unmarshal message: %s", err)
			return err
		}

		logger.Infof("Scraped article: %s, link: %s", article.Title, article.Link)

		// 保存文章信息
		_ = s.Translate(article)
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
		logger.Infof("[%s]Saved article: %s, link: %s", article.Token, article.Title, article.Link)

		return nil
	}))
}

func StartTask() {
	logger.Info("Starting task...")

	// restore articles from storage if exists
	s = storage.NewService()
	err := s.Restore()
	if err != nil {
		panic(err)
	}

	// start scraping
	defer q.Release()
	for _, c := range scrapers {
		if err := c.Run(); err != nil {
			logger.Errorf("Task failed: %s", err)
			return
		}
	}

	q.Wait()
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
		newsaddr.NewBinanceScrapy(q),
	}
}
