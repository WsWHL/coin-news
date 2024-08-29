package utils

import (
	"context"
	"github.com/chromedp/chromedp"
	"news/src/logger"
	"time"
)

func NewBrowserContext(ctx context.Context) (context.Context, context.CancelFunc) {
	options := []chromedp.ExecAllocatorOption{
		chromedp.Headless,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
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
}

func NewGoogleSearch(c context.Context) *GoogleSearch {
	ctx, cancel := NewBrowserContext(c)
	return &GoogleSearch{
		url:    "https://images.google.com/",
		ctx:    ctx,
		cancel: cancel,
	}
}

func (g *GoogleSearch) Search(q string) (string, bool) {
	var (
		url = ""
		ok  = false
	)
	err := chromedp.Run(g.ctx,
		chromedp.Navigate(g.url),
		chromedp.WaitReady("//form//textarea[1]"),
		chromedp.SendKeys("//form//textarea[1]", q),
		chromedp.Submit("//form//button"),
		chromedp.WaitVisible("//div[@id=\"search\"]/div/div/div/div[1]/div/div/div/div[1]/div/h3/a//img"),
		chromedp.Click("//div[@id=\"search\"]/div/div/div/div[1]/div/div/div/div[1]/div/h3/a"),
		chromedp.Sleep(2*time.Second),
		chromedp.AttributeValue("//*[@role=\"dialog\"]/div/div[2]/c-wiz/div/div[3]/div[1]/a/img[1]", "src", &url, &ok),
	)
	if err != nil {
		logger.Errorf("Error searching Google images: %v", err)
		return "", false
	}

	logger.Infof("Search image for '%s' found at: %s", q, url)
	return url, ok
}

func (g *GoogleSearch) Close() {
	g.cancel()
}
