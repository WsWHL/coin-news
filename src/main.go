package main

import (
	"github.com/robfig/cron/v3"
	"news/src/cmd"
	"news/src/config"
	"news/src/logger"
)

func main() {
	logger.Info("Starting server...")

	// start scrapy task scheduler
	go func() {
		cmd.StartScrapyTask()

		c := cron.New()
		id, err := c.AddFunc(config.Cfg.Scrapy.Crontab, cmd.StartScrapyTask)
		if err != nil {
			panic(err)
		}
		c.Start()
		logger.Infof("Added task with ID: %d", id)
	}()

	// start api server
	cmd.StartAPIServer()

	logger.Info("Server started.")
}
