package newsaddr

import (
	"database/sql"
	"github.com/gocolly/colly"
	"github.com/golang-queue/queue"
	"news/src/models"
	"strings"
	"time"
)

// CoinDeskScrapy coindesk news scraping using Colly
type CoinDeskScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewCoinDeskScrapy(q *queue.Queue) *CoinDeskScrapy {
	return &CoinDeskScrapy{
		name:   "coindesk",
		domain: "https://www.coindesk.com",
		send:   NewQueueWrapper(q),
	}
}

func (c *CoinDeskScrapy) OnDetails(url string) models.Article {
	article := models.Article{}

	s := NewScrapy(url)
	s.OnCallback("header.at-news-header", func(e *colly.HTMLElement) {
		title := e.ChildText("div.at-headline h1")
		author := e.ChildText("div.at-authors span a")
		description := e.ChildText("div.at-subheadline h2")
		pubDate := e.ChildText("div.at-created div span")
		image := e.ChildAttr("div.media > figure > picture > img", "src")

		article.From = c.name
		article.Link = url
		article.Title = title
		article.Author = author
		article.Abstract = description
		article.Image = image

		pubDate = strings.ReplaceAll(pubDate, "p.m.", "PM")
		pubDate = strings.ReplaceAll(pubDate, "a.m.", "AM")
		if t, err := time.Parse("Jan 02, 2006 at 15:04 PM MST", pubDate); err == nil {
			article.PubDate = sql.NullTime{Time: t, Valid: true}
		}
	})

	s.Start()

	return article
}

func (c *CoinDeskScrapy) Run() error {
	s := NewScrapy(c.domain)

	// latest
	s.OnCallback("div.live-wire div[class^=live-wirestyles__Wrapper]", func(e *colly.HTMLElement) {
		link := e.ChildAttr("div[class^=live-wirestyles__Title] a", "href")
		url := e.Request.AbsoluteURL(link)

		article := c.OnDetails(url)
		article.Category = models.LatestCategory
		c.send(article)
	})

	// most reads
	s.OnCallback("div.live-wire div[class^=most-read-articlestyles__Wrapper]", func(e *colly.HTMLElement) {
		link := e.ChildAttr("div[class^=most-read-articlestyles__Title] a", "href")
		url := e.Request.AbsoluteURL(link)

		article := c.OnDetails(url)
		article.Category = models.MostReadsCategory
		c.send(article)
	})

	// opinions
	s.OnCallback("div.opinion div[class^=opinionstyles__Wrapper] div[class^=opinionstyles__Wrapper]", func(e *colly.HTMLElement) {
		link := e.ChildAttr("div[class^=opinionstyles__Title] a", "href")
		url := e.Request.AbsoluteURL(link)

		article := c.OnDetails(url)
		article.Category = models.OpinionsCategory
		c.send(article)
	})

	s.Start()

	return nil
}
