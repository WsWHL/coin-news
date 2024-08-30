package cmd

import (
	"context"
	"encoding/json"
	"github.com/golang-queue/queue"
	"github.com/golang-queue/queue/core"
	"news/src/config"
	"news/src/logger"
	"news/src/models"
	"news/src/newsaddr"
	"news/src/storage"
	"news/src/utils"
	"strings"
	"time"
)

var (
	scrapers []newsaddr.Scraper
	q        *queue.Queue
	s        *storage.Service
)

type pluginFunc func(article *models.Article)

func translateTitle() pluginFunc {
	translator, err := utils.NewTranslate(config.Cfg.Kimi.Prompt)
	if err != nil {
		panic(err)
	}

	return func(article *models.Article) {
		if article.From == "jinse" {
			article.TitleCN = article.Title
		}

		if article.Title != "" && article.TitleCN == "" {
			article.TitleCN, _ = translator.Send(article.Title)
		}
	}
}

func searchImage() pluginFunc {
	g := utils.NewGoogleSearch(context.Background())
	return func(article *models.Article) {
		if article.Image == "" {
			article.Image, _ = g.Search(article.Title)
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
			p(article)
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
	for _, c := range scrapers {
		if err := c.Run(); err != nil {
			logger.Errorf("Task failed: %s", err)
			return
		}
	}

	// wait for all queue tasks to finish
	defer q.Release()
	for q.BusyWorkers() > 0 {
		time.Sleep(time.Second)
	}

	logger.Infof("Queue task finished. submitted tasks: %d, success tasks: %d, failure tasks: %d", q.SubmittedTasks(), q.SuccessTasks(), q.FailureTasks())
	logger.Info("Task started successfully.")
}

func init() {
	q = newQueue(
		translateTitle(),
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
	}
}
