package newsaddr

import (
	"context"
	"crypto/tls"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"github.com/golang-queue/queue"
	"net"
	"net/http"
	"news/src/logger"
	"news/src/models"
	"time"
)

type Scrapy struct {
	c     *colly.Collector
	retry int
	url   string
	hdr   map[string]string
}

func NewScrapy(url string) *Scrapy {
	return NewScrapyWithHeader(url, nil)
}

func NewScrapyWithHeader(url string, hdr map[string]string) *Scrapy {
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
		c:     c,
		retry: 3,
		url:   url,
		hdr:   hdr,
	}
}

func (s *Scrapy) Clone(url string) *Scrapy {
	return &Scrapy{
		c:     s.c.Clone(),
		retry: 3,
		url:   url,
		hdr:   s.hdr,
	}
}

func (s *Scrapy) OnCallback(selector string, f colly.HTMLCallback) {
	s.c.OnHTML(selector, f)
}

func (s *Scrapy) OnResponse(f colly.ResponseCallback) {
	s.c.OnResponse(f)
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
