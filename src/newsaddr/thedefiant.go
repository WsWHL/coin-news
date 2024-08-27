package newsaddr

import (
	"fmt"
	"github.com/gocolly/colly"
	"github.com/golang-queue/queue"
	"news/src/logger"
	"news/src/models"
)

// TheDefiantScrapy thedefiant news scraping using Colly
type TheDefiantScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewTheDefiantScrapy(q *queue.Queue) *TheDefiantScrapy {
	return &TheDefiantScrapy{
		name:   "thedefiant",
		domain: "https://thedefiant.io",
		send:   NewQueueWrapper(q),
	}
}

func (t *TheDefiantScrapy) OnDetails(url string) (models.Article, bool) {
	var (
		article = models.Article{}
		success = false
	)

	s := NewScrapy(url)
	s.OnCallback("article", func(e *colly.HTMLElement) {
		title := e.ChildText("article h1:first-of-type")
		description := e.ChildText("article > div:first-of-type")
		author := e.ChildText("article > div:nth-of-type(2) a")
		image := e.ChildAttr("article img.object-cover", "src")

		article.From = t.name
		article.Title = title
		article.Link = url
		article.Author = author
		article.Abstract = description
		article.Image = e.Request.AbsoluteURL(image)
		success = true
	})
	s.Start()

	return article, success
}

func (t *TheDefiantScrapy) OnNewsList(url string, category models.CategoryTypes) models.ArticleList {
	articles := make([]models.Article, 0, 30)

	s := NewScrapy(url)
	s.OnCallback("section > div:first-of-type > div", func(e *colly.HTMLElement) {
		title := e.ChildText("div:nth-of-type(2) a h3")
		link := e.ChildAttr("div:nth-of-type(2) div a:last-of-type", "href")
		description := e.ChildText("div:nth-of-type(2) div.text-base")
		image := e.ChildAttr("div:nth-of-type(2) img.object-cover", "src")

		articles = append(articles, models.Article{
			From:     t.name,
			Category: category,
			Title:    title,
			Link:     e.Request.AbsoluteURL(link),
			Abstract: description,
			Image:    e.Request.AbsoluteURL(image),
		})
	})
	s.Start()

	return articles
}

func (t *TheDefiantScrapy) Run() error {
	// latest
	url := fmt.Sprintf("%s/latest", t.domain)
	latest := t.OnNewsList(url, models.LatestCategory)
	t.send(latest...)

	// analysis
	url = fmt.Sprintf("%s/news/deep-newz", t.domain)
	analysis := t.OnNewsList(url, models.AnalysisCategory)
	t.send(analysis...)

	// opinions
	url = fmt.Sprintf("%s/news/research-and-opinion", t.domain)
	opinions := t.OnNewsList(url, models.OpinionsCategory)
	t.send(opinions...)

	// featured
	s := NewScrapy(t.domain)
	s.OnCallback("main div.grid > div.flex > div.grid h3 a", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		link = e.Request.AbsoluteURL(link)
		if article, ok := t.OnDetails(link); ok {
			article.Category = models.FeaturedCategory
			t.send(article)
		} else {
			logger.Errorf("Failed to fetch article: %s", link)
		}
	})

	// most reads
	s.OnCallback("main div.grid div.flex div.grid:last-of-type div.flex-row", func(e *colly.HTMLElement) {
		title := e.ChildText("h3 a")
		link := e.ChildAttr("h3 a", "href")
		image := e.ChildAttr("a img", "src")

		link = e.Request.AbsoluteURL(link)
		image = e.Request.AbsoluteURL(image)
		t.send(models.Article{
			From:     t.name,
			Category: models.MostReadsCategory,
			Title:    title,
			Link:     link,
			Image:    image,
		})
	})

	s.Start()

	return nil
}
