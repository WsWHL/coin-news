package newsaddr

import (
	"database/sql"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/tidwall/gjson"
	"news/src/logger"
	"news/src/models"
	"time"
)

type JinSeScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewJinSeScrapy(q QueueWrapper) *JinSeScrapy {
	return &JinSeScrapy{
		name:   "jinse",
		domain: "https://www.jinse.cn",
		send:   q,
	}
}

func (j *JinSeScrapy) OnFeatured(body []byte) models.ArticleList {
	articles := make([]models.Article, 0, 30)

	data := gjson.GetBytes(body, "data.list.#.object_1")
	data.ForEach(func(_, i gjson.Result) bool {
		ts := i.Get("published_at").Int()
		articles = append(articles, models.Article{
			From:     j.name,
			Category: models.FeaturedCategory,
			Title:    i.Get("title").String(),
			Abstract: i.Get("summary").String(),
			Link:     i.Get("jump_url").String(),
			Image:    i.Get("cover").String(),
			Reads:    int(i.Get("show_read_number").Int()),
			Author:   i.Get("author.nickname").String(),
			PubDate: sql.NullTime{
				Time:  time.Unix(ts, 0),
				Valid: true,
			},
		})
		return true
	})

	return articles
}

func (j *JinSeScrapy) OnNewsAPI(body []byte, category models.CategoryTypes) models.ArticleList {
	articles := make([]models.Article, 0, 30)

	data := gjson.GetBytes(body, "data")
	data.ForEach(func(_, i gjson.Result) bool {
		ts := i.Get("published_at").Int()
		article := models.Article{
			From:     j.name,
			Category: category,
			Title:    i.Get("title").String(),
			Link:     i.Get("jump_url").String(),
			PubDate: sql.NullTime{
				Time:  time.Unix(ts, 0),
				Valid: true,
			},
		}
		if category == models.MostReadsCategory {
			article.Image = fmt.Sprintf("%s_small.png", i.Get("covers").String())
			article.Reads = int(i.Get("read_number").Int())
			article.Author = i.Get("author.nickname").String()
		}

		articles = append(articles, article)
		return true
	})

	return articles
}

func (j *JinSeScrapy) Run() error {
	// featured
	s := NewScrapy("https://api.jinse.cn/noah/v3/timelines?catelogue_key=www&limit=30")
	s.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		featured := j.OnFeatured(r.Body)
		j.send(featured...)
	})
	s.Start()

	// latest
	s1 := s.Clone("https://newapi.jinse.cn/noah/v1/breaking-news")
	s1.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		latest := j.OnNewsAPI(r.Body, models.LatestCategory)
		j.send(latest...)
	})
	s1.Start()

	// most reads
	s2 := s1.Clone("https://newapi.jinse.cn/noah/v1/articles/hot?hour_diff=24")
	s2.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			logger.Errorf("Response status code: %d", r.StatusCode)
			return
		}

		mostReads := j.OnNewsAPI(r.Body, models.MostReadsCategory)
		j.send(mostReads...)
	})
	s2.Start()

	return nil
}
