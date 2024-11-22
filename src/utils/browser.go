package utils

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"news/src/config"
	"news/src/logger"
	"sync"
	"time"
)

func NewBrowserContext(ctx context.Context) (context.Context, context.CancelFunc) {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Headless,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.UserAgent(config.Cfg.Scrapy.UA),
	}
	c, _ := chromedp.NewExecAllocator(ctx, options...)

	return chromedp.NewContext(c,
		chromedp.WithLogf(logger.Infof),
		chromedp.WithErrorf(logger.Errorf),
	)
}

type GoogleSearch struct {
	url    string
	ctx    context.Context
	cancel context.CancelFunc
	lock   sync.Mutex
}

func NewGoogleSearch(c context.Context) *GoogleSearch {
	ctx, cancel := NewBrowserContext(c)
	return &GoogleSearch{
		url:    "https://images.google.com/",
		ctx:    ctx,
		cancel: cancel,
		lock:   sync.Mutex{},
	}
}

func (g *GoogleSearch) Search(q string) (string, bool) {
	g.lock.Lock()
	defer g.lock.Unlock()

	var (
		url = ""
		ok  = false
	)
	err := chromedp.Run(g.ctx,
		chromedp.Navigate(g.url),
		chromedp.WaitReady("//form//textarea[1]"),
		chromedp.SendKeys("//form//textarea[1]", q),
		chromedp.Submit("//form//button"),
		chromedp.WaitVisible("//div[@id=\"search\"]/div/div/div/div/div[1]/div/div/div[1]/div/h3/a//img"),
		chromedp.DoubleClick("//div[@id=\"search\"]/div/div/div/div/div[1]/div/div/div[1]"),
		chromedp.WaitVisible("//*[@role=\"dialog\"]/div/div[2]/c-wiz/div/div[3]/div[1]/a/img[1]"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var count int
			for i := 0; i < 10; i++ {
				time.Sleep(time.Second)

				err := chromedp.Evaluate("document.querySelectorAll('div[role=\"dialog\"] > div > div:nth-of-type(2) > c-wiz a > img[jsaction]').length", &count).Do(ctx)
				if err == nil && count > 0 {
					logger.Infof("Found image try %d times", i+1)
					err = chromedp.AttributeValue("//*[@role=\"dialog\"]/div/div[2]/c-wiz/div/div[3]/div[1]/a/img[@jsaction]", "src", &url, &ok).Do(ctx)
					if err != nil {
						return err
					}
					break
				}
			}

			if url == "" {
				err := chromedp.AttributeValue("//*[@role=\"dialog\"]/div/div[2]/c-wiz/div/div[3]/div[1]/a/img[1]", "src", &url, &ok).Do(ctx)
				if err != nil {
					return err
				}
			}

			for i := 2; len(url) > 512; i++ {
				if err := chromedp.DoubleClick(fmt.Sprintf("//div[@id=\"search\"]/div/div/div/div/div[1]/div/div/div[%d]", i)).Do(ctx); err != nil {
					return err
				}
				time.Sleep(time.Second)
				_ = chromedp.AttributeValue("//*[@role=\"dialog\"]/div/div[2]/c-wiz/div/div[3]/div[1]/a/img[1]", "src", &url, &ok).Do(ctx)
			}

			return nil
		}),
	)
	if err != nil {
		return "", false
	}

	logger.Infof("Search image for '%s' found at: %s", q, url)
	return url, ok
}

func (g *GoogleSearch) Close() {
	g.cancel()
}
