package newsaddr

import (
	"database/sql"
	"github.com/gocolly/colly"
	"news/src/models"
	"strconv"
	"strings"
	"time"
)

type BitPieScrapy struct {
	name   string
	domain string
	send   QueueWrapper
}

func NewBitPieScrapy(q QueueWrapper) *BitPieScrapy {
	return &BitPieScrapy{
		name:   "bitpie",
		domain: "https://m.sc5b.net",
		send:   q,
	}
}

func (b *BitPieScrapy) Run() error {
	s := NewScrapy(b.domain)

	// featured
	s.OnCallback("div.home-main article.picsrcd div.entry-container", func(e *colly.HTMLElement) {
		title := e.ChildAttr("header h3 a", "title")
		link := e.ChildAttr("header h3 a", "href")
		description := e.ChildText("div.entry-summary p")
		author := e.ChildText("div.entry-meta-items div.entry-meta-author a")
		image := e.ChildAttr("figure.block-image a img", "src")
		date := e.ChildAttr("div.entry-meta-items time", "datetime")
		readsText := e.ChildAttr("div.entry-meta-items span.meta-viewnums", "title")
		pubDate, _ := time.Parse("2006-01-02 15:04:05", date)
		reads, _ := strconv.Atoi(strings.Split(readsText, " ")[0])

		b.send(models.Article{
			From:     b.name,
			Category: models.FeaturedCategory,
			Title:    title,
			Link:     link,
			Author:   author,
			Image:    image,
			Abstract: description,
			PubDate:  sql.NullTime{Time: pubDate, Valid: true},
			Reads:    reads,
		})
	})

	// latest
	var author string
	s.OnCallback("section.widget_avatar", func(e *colly.HTMLElement) {
		author = e.ChildAttr("div.user-bgif img", "title")
	})
	s.OnCallback("section#divPrevious ul.divPrevious div.side_new", func(e *colly.HTMLElement) {
		title := e.ChildText("div.side-new-title a")
		link := e.ChildAttr("div.side-new-title a", "href")
		date := e.ChildText("div.side-new-time")[6:]
		pubDate, _ := time.Parse("2006月01月02日", date)

		b.send(models.Article{
			From:     b.name,
			Category: models.LatestCategory,
			Title:    title,
			Link:     link,
			Author:   author,
			PubDate:  sql.NullTime{Time: pubDate, Valid: true},
		})
	})

	s.Start()

	return nil
}
