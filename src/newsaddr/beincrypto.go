package newsaddr

import (
	"database/sql"
	"fmt"
	"github.com/gocolly/colly"
	"news/src/logger"
	"news/src/models"
	"strings"
	"time"
)

// BeinCryptoScrapy beincrypt news scraping using Colly
type BeinCryptoScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewBeinCryptoScrapy(q QueueWrapper) *BeinCryptoScrapy {
	return &BeinCryptoScrapy{
		name:   "beincrypto",
		domain: "https://www.beincrypto.com",
		send:   q,
	}
}

func (b *BeinCryptoScrapy) OnDetails(url string) (models.Article, bool) {
	var (
		article = models.Article{}
		success = false
	)

	s := NewBrowserScrapy(url)
	s.OnCallback("article div[data-el='main-content']", func(e *colly.HTMLElement) {
		title := e.ChildText("header h1")
		image := e.ChildAttr("div.featured-images figure img.bic-featured", "src")
		author := e.ChildText("div[data-el='bic-author-meta'] a span")
		description := e.ChildText("ul.in-brief-block li:nth-of-type(1)")
		pubDate := e.ChildAttr("time", "datetime")

		article.From = b.name
		article.Title = title
		article.Author = author
		article.Image = image
		article.Link = url
		article.Abstract = description
		if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
			article.PubDate = sql.NullTime{Time: t, Valid: true}
		}

		success = true
	})
	s.Start()

	return article, success
}

func (b *BeinCryptoScrapy) OnList(path string, category models.CategoryTypes) models.ArticleList {
	articles := make([]models.Article, 0, 30)

	url := fmt.Sprintf("%s%s", b.domain, path)
	s := NewBrowserScrapy(url)
	s.OnCallback("main#bic-main-content > div:nth-of-type(3) > div", func(e *colly.HTMLElement) {
		title := e.ChildText("h5 a")
		link := e.ChildAttr("h5 a", "href")
		images := e.ChildAttr("div[data-el='bic-c-card-image'] a img", "data-srcset")
		date := e.ChildAttr("time", "datetime")
		image := strings.Split(strings.Split(images, ",")[1], " ")[1]

		var pubDate time.Time
		if t, err := time.Parse(time.RFC3339, date); err == nil {
			pubDate = t
		}
		articles = append(articles, models.Article{
			From:     b.name,
			Category: category,
			Title:    title,
			Link:     link,
			Image:    image,
			PubDate:  sql.NullTime{Time: pubDate, Valid: true},
		})
	})
	s.Start()

	return articles
}

func (b *BeinCryptoScrapy) Run() error {
	// latest
	latest := b.OnList("/news/", models.LatestCategory)
	b.send(latest...)

	// analysis
	analysis := b.OnList("/analysis/", models.AnalysisCategory)
	b.send(analysis...)

	// opinions
	opinions := b.OnList("/opinion/", models.OpinionsCategory)
	b.send(opinions...)

	// featured
	s := NewBrowserScrapy(b.domain)
	s.OnCallback("main#bic-main-content section:nth-of-type(1) > div > div:nth-of-type(1)", func(e *colly.HTMLElement) {
		title := e.ChildAttr("figure a", "title")
		link := e.ChildAttr("figure a", "href")

		if article, success := b.OnDetails(link); success {
			article.Category = models.FeaturedCategory
			b.send(article)
		} else {
			logger.Errorf("Failed to scrape article: title: %s, url: %s", title, link)
		}
	})

	s.OnCallback("main#bic-main-content section:nth-of-type(1) > div > div:nth-of-type(2) ul li", func(e *colly.HTMLElement) {
		title := e.ChildText("a")
		link := e.ChildAttr("a", "href")

		if article, success := b.OnDetails(link); success {
			article.Category = models.FeaturedCategory
			b.send(article)
		} else {
			logger.Errorf("Failed to scrape article: title: %s, url: %s", title, link)
		}
	})

	// most reads
	s.OnCallback("main#bic-main-content section:nth-of-type(1) > div > div:nth-of-type(3) ul li", func(e *colly.HTMLElement) {
		title := e.ChildText("a")
		link := e.ChildAttr("a", "href")

		if article, success := b.OnDetails(link); success {
			article.Category = models.MostReadsCategory
			b.send(article)
		} else {
			logger.Errorf("Failed to scrape article: title: %s, url: %s", title, link)
		}
	})

	s.Start()

	return nil
}
