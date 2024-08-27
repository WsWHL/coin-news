package newsaddr

import (
	"database/sql"
	"github.com/gocolly/colly"
	"github.com/golang-queue/queue"
	"news/src/models"
	"strings"
	"time"
)

// TheBlockScrapy theblock news scraping using Colly
type TheBlockScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewTheBlockScrapy(q *queue.Queue) *TheBlockScrapy {
	return &TheBlockScrapy{
		name:   "theblock",
		domain: "https://www.theblock.co",
		send:   NewQueueWrapper(q),
	}
}

func (b *TheBlockScrapy) OnDetails(url string) models.Article {
	article := models.Article{}

	s := NewScrapy(url)
	s.OnCallback("article.articleBody", func(e *colly.HTMLElement) {
		title := e.ChildText("h1[class^=articleLabel]")
		author := e.ChildText("div.articleByline a")
		image := e.ChildAttr("div.articleFeatureImage img", "src")
		pubDate := e.ChildText("div.ArticleTimestamps div.ArticleTimestamps__container")
		description := e.ChildText("div.quickTake ul li:nth-of-type(1) span")

		pubDate = strings.Split(pubDate, "•")[1][1:]
		if t, err := time.Parse("January 02, 2006, 15:04PM MST", pubDate); err == nil {
			article.PubDate = sql.NullTime{Time: t, Valid: true}
		}

		article.From = b.name
		article.Link = url
		article.Title = title
		article.Author = author
		article.Image = image
		article.Abstract = description
	})

	s.Start()

	return article
}

func (b *TheBlockScrapy) Run() error {
	s := NewScrapy(b.domain)

	s.OnCallback("div.heroLeftRail div.latestNews article", func(e *colly.HTMLElement) {
		link := e.ChildAttr("div.textCard__content a.textCard__link", "href")
		url := e.Request.AbsoluteURL(link)

		article := b.OnDetails(url)
		article.Category = models.LatestCategory
		b.send(article)
	})

	s.OnCallback("div.featuredStories article", func(e *colly.HTMLElement) {
		title := e.ChildText("div[class$=__content] a > h2")
		link := e.ChildAttr("div[class$=__content] a.appLink", "href")
		image := e.ChildAttr("a > img[class$=image]", "src")
		date := e.ChildText("div.meta__timestamp")

		pubDate := sql.NullTime{}
		if t, err := time.Parse("January 02, 2006, 15:04PM MST", date); err == nil {
			pubDate.Time = t
			pubDate.Valid = true
		}

		b.send(models.Article{
			From:     b.name,
			Category: models.FeaturedCategory,
			Title:    title,
			Link:     e.Request.AbsoluteURL(link),
			Image:    image,
			PubDate:  pubDate,
		})
	})

	s.Start()

	return nil
}
