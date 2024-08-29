package newsaddr

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/golang-queue/queue"
	"net"
	"net/http"
	"net/url"
	"news/src/logger"
	"news/src/models"
	"news/src/utils"
	"time"
)

type Scrapy struct {
	c             *colly.Collector
	retry         int
	url           string
	hdr           map[string]string
	htmlCallbacks []htmlCallbackContainer
	respCallbacks []colly.ResponseCallback
}

func NewScrapy(url string) *Scrapy {
	c := colly.NewCollector(
		colly.UserAgent(browser.Chrome()),
		colly.AllowURLRevisit(),
		colly.Debugger(&debug.LogDebugger{}),
	)
	c.WithTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: defaultDialContext(&net.Dialer{
			Timeout:   180 * time.Second,
			KeepAlive: 60 * time.Second,
		}),
		ForceAttemptHTTP2:     true,
		ResponseHeaderTimeout: 60 * time.Second,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       getCloudFlareTLSConfiguration(),
	})

	return &Scrapy{
		c:             c,
		retry:         3,
		url:           url,
		htmlCallbacks: make([]htmlCallbackContainer, 0),
		respCallbacks: make([]colly.ResponseCallback, 0),
	}
}

func (s *Scrapy) WithHeader(hdr map[string]string) *Scrapy {
	s.hdr = hdr
	return s
}

func (s *Scrapy) Clone(url string) *Scrapy {
	return &Scrapy{
		c:             s.c.Clone(),
		retry:         3,
		url:           url,
		hdr:           s.hdr,
		htmlCallbacks: make([]htmlCallbackContainer, 0),
		respCallbacks: make([]colly.ResponseCallback, 0),
	}
}

func (s *Scrapy) OnCallback(selector string, f colly.HTMLCallback) {
	s.c.OnHTML(selector, f)
	s.htmlCallbacks = append(s.htmlCallbacks, htmlCallbackContainer{selector, f})
}

func (s *Scrapy) OnResponse(f colly.ResponseCallback) {
	s.c.OnResponse(f)
	s.respCallbacks = append(s.respCallbacks, f)
}

func (s *Scrapy) Start() {
	s.c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("User-Agent", browser.Chrome())

		// Set custom headers
		for k, v := range s.hdr {
			r.Headers.Set(k, v)
		}

		logger.Infof("Visiting: %s", r.URL)
	})

	// Retry up to 3 times if the request fails
	count := s.retry
	s.c.OnScraped(func(r *colly.Response) {
		logger.Infof("Finished: %s", r.Request.URL)
		count = s.retry // reset retry count
	})

	s.c.OnError(func(r *colly.Response, err error) {
		logger.Errorf("Request URL: %s, status_code: %d, error: %s", r.Request.URL, r.StatusCode, err)

		// If the request fails, using browser scraping retry the request
		if r.StatusCode == http.StatusForbidden {
			b := NewBrowserScrapyFromColly(s, r.Request.URL.String())
			b.Start()
			return
		}

		count--
		for count >= 0 {
			logger.Infof("Retrying %d more times...", s.retry-count)
			time.Sleep(time.Second * 1)
			if err = r.Request.Retry(); err == nil {
				return
			}
		}
	})

	if err := s.c.Visit(s.url); err != nil {
		logger.Errorf("url: %s, error: %s", s.url, err)
	}
}

type BrowserScrapy struct {
	url    string
	ctx    context.Context
	cancel context.CancelFunc

	htmlCallbacks []htmlCallbackContainer
	respCallbacks []colly.ResponseCallback
}

type htmlCallbackContainer struct {
	Selector string
	Function colly.HTMLCallback
}

func NewBrowserScrapy(url string) *BrowserScrapy {
	ctx, cancel := utils.NewBrowserContext(context.Background())
	return &BrowserScrapy{
		url:           url,
		ctx:           ctx,
		cancel:        cancel,
		htmlCallbacks: make([]htmlCallbackContainer, 0),
		respCallbacks: make([]colly.ResponseCallback, 0),
	}
}

func NewBrowserScrapyFromColly(c *Scrapy, url string) *BrowserScrapy {
	ctx, cancel := utils.NewBrowserContext(context.Background())
	return &BrowserScrapy{
		url:           url,
		ctx:           ctx,
		cancel:        cancel,
		htmlCallbacks: c.htmlCallbacks,
		respCallbacks: c.respCallbacks,
	}
}

func (b *BrowserScrapy) OnCallback(selector string, f colly.HTMLCallback) {
	b.htmlCallbacks = append(b.htmlCallbacks, htmlCallbackContainer{selector, f})
}

func (b *BrowserScrapy) OnResponse(f colly.ResponseCallback) {
	b.respCallbacks = append(b.respCallbacks, f)
}

func (b *BrowserScrapy) Start() {
	defer b.cancel()

	var html string
	var jsonText string
	resp, err := chromedp.RunResponse(b.ctx,
		chromedp.Navigate(b.url),
		chromedp.OuterHTML("html", &html),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Evaluate("document.body.innerText", &jsonText).Do(ctx)
		}),
	)
	if err != nil {
		logger.Errorf("Failed to start browser: %s", err)
		return
	}

	req := &colly.Request{}
	req.URL, _ = url.Parse(b.url)
	switch resp.MimeType {
	case "text/html":
		html = fmt.Sprintf("<!DOCTYPE html>\n%s", html)
		doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(html))
		if err != nil {
			logger.Errorf("Failed to parse HTML: %s", err)
			return
		}

		for _, c := range b.htmlCallbacks {
			i := 0
			doc.Find(c.Selector).Each(func(_ int, s *goquery.Selection) {
				for _, n := range s.Nodes {
					e := colly.NewHTMLElementFromSelectionNode(&colly.Response{
						Request:    req,
						StatusCode: int(resp.Status),
					}, s, n, i)
					i++
					c.Function(e)
				}
			})
		}
	case "application/json":
		for _, r := range b.respCallbacks {
			r(&colly.Response{
				StatusCode: int(resp.Status),
				Body:       []byte(jsonText),
				Request:    req,
			})
		}
	}
}

func defaultDialContext(dialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return dialer.DialContext
}

// getCloudFlareTLSConfiguration returns an accepted client TLS configuration to not get detected by CloudFlare directly
// in case the configuration needs to be updated later on: https://wiki.mozilla.org/Security/Server_Side_TLS .
func getCloudFlareTLSConfiguration() *tls.Config {
	return &tls.Config{
		CurvePreferences: []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521, tls.X25519},
	}
}

type Scraper interface {
	Run() error
}

// QueueWrapper is a function that takes a list of articles and sends them to a queue.
type QueueWrapper func(articles ...models.Article)

// NewQueueWrapper wraps a queue to send articles to it.
func NewQueueWrapper(q *queue.Queue) QueueWrapper {
	return func(articles ...models.Article) {
		for _, article := range articles {
			if err := q.Queue(&article); err != nil {
				logger.Errorf("Failed to send article: %s", err)
				continue
			}
		}
	}
}
