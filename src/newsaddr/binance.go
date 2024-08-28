package newsaddr

import (
	"database/sql"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/golang-queue/queue"
	"github.com/tidwall/gjson"
	"news/src/logger"
	"news/src/models"
	"time"
)

// BinanceScrapy binance news scraping using Colly
type BinanceScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewBinanceScrapy(q *queue.Queue) *BinanceScrapy {
	return &BinanceScrapy{
		name:   "binance",
		domain: "https://www.binance.com",
		send:   NewQueueWrapper(q),
	}
}

func (b *BinanceScrapy) OnListAPI(body []byte, category models.CategoryTypes) models.ArticleList {
	articles := make([]models.Article, 0, 30)

	data := gjson.GetBytes(body, "data.vos")
	data.ForEach(func(_, i gjson.Result) bool {
		t := time.Unix(i.Get("date").Int(), 0)
		articles = append(articles, models.Article{
			From:         b.name,
			Category:     category,
			Title:        i.Get("title").String(),
			Author:       i.Get("authorName").String(),
			Abstract:     i.Get("subTitle").String(),
			Link:         i.Get("webLink").String(),
			Image:        i.Get("coverMeta.url").String(),
			PubDate:      sql.NullTime{Time: t, Valid: true},
			Reads:        int(i.Get("viewCount").Int()),
			Interactions: int(i.Get("likeCount").Int()),
			Comments:     int(i.Get("commentCount").Int()),
		})
		return true
	})

	return articles
}

func (b *BinanceScrapy) Run() error {
	// most reads
	url := fmt.Sprintf("%s/bapi/composite/v3/friendly/pgc/content/article/list?pageIndex=1&pageSize=30&type=1", b.domain)
	s := NewScrapy(url).WithHeader(map[string]string{
		"content-type": "application/json",
		"clienttype":   "web",
		"lang":         "en-US",
	})
	s.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		list := b.OnListAPI(r.Body, models.MostReadsCategory)
		b.send(list...)
		logger.Infof("Binance most reads news scraped successfully.")
	})
	s.Start()

	// latest
	url = fmt.Sprintf("%s/bapi/composite/v4/friendly/pgc/feed/news/list?pageIndex=1&pageSize=30&strategy=6&tagId=0&featured=false", b.domain)
	s1 := s.Clone(url)
	s1.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		list := b.OnListAPI(r.Body, models.LatestCategory)
		b.send(list...)
		logger.Infof("Binance latest news scraped successfully.")
	})
	s1.Start()

	return nil
}
