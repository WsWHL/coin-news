package newsaddr

import (
	"database/sql"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/golang-queue/queue"
	"news/src/logger"
	"news/src/models"
	"strings"
	"time"
)

// BlockWorksScrapy Blockworks news scraping using Colly
type BlockWorksScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewBlockWorksScrapy(q *queue.Queue) *BlockWorksScrapy {
	return &BlockWorksScrapy{
		name:   "blockworks",
		domain: "https://blockworks.co",
		send:   NewQueueWrapper(q),
	}
}

func (b *BlockWorksScrapy) OnDetails(url string) models.Article {
	article := models.Article{}

	s := NewScrapy(url)
	s.OnCallback("article", func(e *colly.HTMLElement) {
		article.Title = e.ChildText("h1:first-of-type")
		article.Abstract = e.ChildText("div:first-of-type > p.text-left")
		author := e.ChildText("div:first-of-type div.uppercase:first-of-type")
		pubDate := e.ChildAttr("div:first-of-type div.uppercase:last-of-type time", "datetime")
		image := e.ChildAttr("div:nth-of-type(2) img.object-cover", "src")

		article.Image = e.Request.AbsoluteURL(image)
		article.Author = author[3 : len(author)-2]
		if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
			article.PubDate = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		}
	})
	s.Start()

	return article
}

func (b *BlockWorksScrapy) OnHomepageNews() (models.ArticleList, models.ArticleList) {
	latest := make([]models.Article, 0, 30)
	featured := make([]models.Article, 0, 30)

	s := NewScrapy(b.domain)

	// latest news
	s.OnCallback("section.flex section", func(e *colly.HTMLElement) {
		link := e.ChildAttr("div:nth-child(2) > a", "href")
		link = e.Request.AbsoluteURL(link)

		article := b.OnDetails(link)
		article.From = b.name
		article.Category = models.LatestCategory
		article.Link = link
		if article.Title != "" {
			latest = append(latest, article)
		}
	})

	// featured
	s.OnCallback("section.flex div.order-1 > div", func(e *colly.HTMLElement) {
		e.ForEach("div.flex.justify-center.items-start.self-stretch.gap-3:nth-child(1)", func(i int, c *colly.HTMLElement) {
			article := models.Article{}
			article.From = b.name
			article.Category = models.FeaturedCategory
			article.Title = c.ChildText("div > div:nth-child(2) > a")
			article.Abstract = c.ChildText("div > div:nth-child(3) > p")
			link := c.ChildAttr("div > div:nth-child(2) > a", "href")
			pubDate := c.ChildAttr("div > div:nth-child(4) > div > time", "datetime")
			image := c.ChildAttr("div > div:nth-child(5) img[alt=article-image]", "src")

			authors := make([]string, 0)
			c.ForEach("div > div:nth-child(4) > div > span a.link-gray", func(_ int, i *colly.HTMLElement) {
				authors = append(authors, i.Text)
			})
			article.Author = strings.Join(authors, " & ")
			article.Link = c.Request.AbsoluteURL(link)
			article.Image = c.Request.AbsoluteURL(image)

			if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
				article.PubDate = sql.NullTime{
					Time:  t,
					Valid: true,
				}
			}
			featured = append(featured, article)
		})

		e.ForEach("div.justify-start.items-center.flex-grow.gap-2.w-full", func(i int, c *colly.HTMLElement) {
			article := models.Article{}
			article.From = b.name
			article.Category = models.FeaturedCategory
			article.Title = c.ChildText("div > div:nth-of-type(2) > a")
			article.Abstract = c.ChildText("div > div:nth-of-type(3)")
			article.Author = c.ChildText("div > div:nth-of-type(4) > div > span a.link-gray")
			link := c.ChildAttr("div > div:nth-of-type(2) > a", "href")
			pubDate := c.ChildAttr("div > div:nth-of-type(4) > div > time", "datetime")
			image := c.ChildAttr("div > a > img[alt=article-image]", "src")

			authors := make([]string, 0)
			c.ForEach("div > div:nth-of-type(4) > div > span a.link-gray", func(_ int, i *colly.HTMLElement) {
				authors = append(authors, i.Text)
			})
			article.Author = strings.Join(authors, " & ")
			article.Link = c.Request.AbsoluteURL(link)
			if image != "" {
				article.Image = c.Request.AbsoluteURL(image)
			} else {
				article.Image = b.OnDetails(article.Link).Image
			}

			if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
				article.PubDate = sql.NullTime{
					Time:  t,
					Valid: true,
				}
			}
			featured = append(featured, article)
		})
	})

	s.Start()

	return latest, featured
}

func (b *BlockWorksScrapy) OnOpinionNews() models.ArticleList {
	articles := make([]models.Article, 0, 30)

	s := NewScrapy(fmt.Sprintf("%s/category/opinion", b.domain))
	s.OnCallback("section.flex div.flex.flex-col.justify-start.self-stretch.flex-grow.gap-2.w-full", func(e *colly.HTMLElement) {
		title := e.ChildText("div:nth-child(3) > a")
		description := e.ChildText("div:nth-child(4) > p")
		author := e.ChildText("div:nth-child(5) > div > span > a")
		pubDate := e.ChildAttr("div:nth-child(5) > div > time", "datetime")
		link := e.ChildAttr("a.cursor-pointer", "href")
		image := e.ChildAttr("a > img[alt=article-image]", "src")

		link = e.Request.AbsoluteURL(link)
		image = e.Request.AbsoluteURL(image)
		if title == "" || link == "" {
			logger.Errorf("article data is missing. Skipping article: %s", e.DOM.Text())
			return
		}

		article := models.Article{
			From:     b.name,
			Category: models.OpinionsCategory,
			Title:    title,
			Author:   author,
			Link:     link,
			Image:    image,
			Abstract: description,
		}
		if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
			article.PubDate = sql.NullTime{
				Time:  t,
				Valid: true,
			}
		}

		articles = append(articles, article)
	})
	s.Start()

	return articles
}

func (b *BlockWorksScrapy) Run() error {
	// latest and featured articles
	latest, featured := b.OnHomepageNews()
	b.send(latest...)
	b.send(featured...)

	// opinion articles
	opinions := b.OnOpinionNews()
	b.send(opinions...)

	return nil
}
