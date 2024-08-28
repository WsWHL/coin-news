package newsaddr

import (
	"database/sql"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/golang-queue/queue"
	"github.com/tidwall/gjson"
	"news/src/logger"
	"news/src/models"
	"strings"
	"time"
)

// DecryptScrapy decrypt news scraping using Colly
type DecryptScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewDecryptScrapy(q *queue.Queue) *DecryptScrapy {
	return &DecryptScrapy{
		name:   "decrypt",
		domain: "https://decrypt.co",
		send:   NewQueueWrapper(q),
	}
}

func (d *DecryptScrapy) GetBuildId() string {
	var buildId string

	s := NewScrapy(d.domain)
	s.OnCallback("head script[src$='_buildManifest.js']", func(e *colly.HTMLElement) {
		buildId = strings.Split(e.Attr("src"), "/")[3]
	})

	s.Start()

	return buildId
}

func (d *DecryptScrapy) OnNewsAPI(body []byte, category models.CategoryTypes) models.ArticleList {
	articles := make(models.ArticleList, 0, 30)

	data := gjson.GetBytes(body, "pageProps.dehydratedState.queries.#(state.data.pages.#(articles.data.#(__typename==\"NewsArticleEntity\")))#.state.data.pages.0.articles.data")
	if data.IsArray() {
		data = data.Array()[0]
		data.ForEach(func(_, i gjson.Result) bool {
			date := i.Get("publishedAt").String()
			pubDate, _ := time.Parse("2006-01-02T15:04:05", date)

			articles = append(articles, models.Article{
				From:     d.name,
				Category: category,
				Title:    i.Get("title").String(),
				Abstract: i.Get("blurb").String(),
				Image:    i.Get("featuredImage.src").String(),
				Author:   i.Get("authors.data.0.name").String(),
				Link:     fmt.Sprintf("%s%s", d.domain, i.Get("meta.hreflangs.0.path").String()),
				PubDate:  sql.NullTime{Time: pubDate, Valid: true},
			})
			return true
		})
	}

	return articles
}

func (d *DecryptScrapy) OnCoinAPI(buildId string, body []byte) {
	slugs := gjson.GetBytes(body, "pageProps.priceQuotes.#.slug").Array()
	if len(slugs) > 30 {
		slugs = slugs[:30]
	}

	index := 0
	s := NewScrapy(fmt.Sprintf("%s/_next/data/%s/en-US/price/%s.json", d.domain, buildId, slugs[index]))
	s.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
		}

		data := gjson.GetBytes(r.Body, "pageProps.dehydratedState.queries.#(state.data.pages.#(articles.data.#(__typename==\"NewsArticleEntity\")))#.state.data.pages.0.articles.data")
		if data.IsArray() {
			data = data.Array()[0]
			data.ForEach(func(_, i gjson.Result) bool {
				date := i.Get("publishedAt").String()
				pubDate, _ := time.Parse("2006-01-02T15:04:05", date)

				d.send(models.Article{
					From:     d.name + "_coin",
					Category: models.CategoryTypes(slugs[index].String()),
					Title:    i.Get("title").String(),
					Abstract: i.Get("blurb").String(),
					Image:    i.Get("featuredImage.src").String(),
					Author:   i.Get("authors.data.0.name").String(),
					Link:     fmt.Sprintf("%s%s", d.domain, i.Get("meta.hreflangs.0.path").String()),
					PubDate:  sql.NullTime{Time: pubDate, Valid: true},
				})
				return true
			})
		}

		index++
		if index < len(slugs) {
			err := r.Request.Visit(fmt.Sprintf("%s/_next/data/%s/en-US/price/%s.json", d.domain, buildId, slugs[index]))
			if err != nil {
				logger.Errorf("Next request failed: %v", err)
			}
		}
	})

	s.Start()
}

func (d *DecryptScrapy) Run() error {
	buildId := d.GetBuildId()
	logger.Infof("Decrypt Build ID: %s", buildId)

	// coin prices
	url := fmt.Sprintf("%s/_next/data/%s/en-US/degen-alley.json", d.domain, buildId)
	c := NewScrapy(url)
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		d.OnCoinAPI(buildId, r.Body)
	})
	c.Start()

	// latest
	url = fmt.Sprintf("%s/_next/data/%s/en-US/news.json?parent_term_slug=news", d.domain, buildId)
	s := NewScrapy(url)
	s.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		latest := d.OnNewsAPI(r.Body, models.LatestCategory)
		d.send(latest...)
	})
	s.Start()

	// featured
	url = fmt.Sprintf("%s/_next/data/%s/en-US/news/editors-picks.json?parent_term_slug=news&term_slug=editors-picks", d.domain, buildId)
	s1 := s.Clone(url)
	s1.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		featured := d.OnNewsAPI(r.Body, models.FeaturedCategory)
		d.send(featured...)
	})
	s1.Start()

	// opinions
	url = fmt.Sprintf("%s/_next/data/%s/en-US/news/opinion.json?parent_term_slug=news&term_slug=opinion", d.domain, buildId)
	s2 := s1.Clone(url)
	s2.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		opinions := d.OnNewsAPI(r.Body, models.OpinionsCategory)
		d.send(opinions...)
	})
	s2.Start()

	return nil
}
