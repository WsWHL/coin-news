package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Article 文章信息
type Article struct {
	ID           int           `gorm:"column:id;primaryKey" json:"id"`
	Token        string        `gorm:"column:token;size:256;index:idx_token" json:"token"`
	From         string        `gorm:"column:from;size:64;idx_from" json:"from"`
	Title        string        `gorm:"column:title;size:256;index:idx_title;not null" json:"title"`
	TitleCN      string        `gorm:"column:title_ch;size:256" json:"title_cn"`
	Abstract     string        `gorm:"column:abstract;type:text" json:"abstract"`
	Image        string        `gorm:"column:image;size:512" json:"image"`
	Link         string        `gorm:"column:link;size:512" json:"link"`
	PubDate      sql.NullTime  `gorm:"column:pub_date" json:"pub_date"`
	Author       string        `gorm:"column:author;size:64" json:"author"`
	Category     CategoryTypes `gorm:"column:category;size:64;index:idx_category" json:"category"`
	Reads        int           `gorm:"column:reads" json:"reads"`
	Interactions int           `gorm:"column:interactions" json:"interactions"`
	Comments     int           `gorm:"column:comments" json:"comments"`
	Notes        string        `gorm:"column:notes;size:256" json:"notes"`
	CreateTime   time.Time     `gorm:"column:create_time" json:"create_time"`
	UpdateTime   time.Time     `gorm:"column:update_time" json:"update_time"`
}

func (a *Article) TableName() string {
	return "articles"
}

func (a *Article) MarshalBinary() ([]byte, error) {
	return json.Marshal(a)
}

func (a *Article) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, a)
}

func (a *Article) Bytes() []byte {
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}

	return b
}

func (a *Article) GenToken() string {
	if a.Token != "" {
		return a.Token
	}

	data := fmt.Sprintf("%s", a.Title)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (a *Article) GetTitleByLang(lang string) string {
	if lang == "ch" {
		return a.TitleCN
	}

	return a.Title
}

func (a *Article) GetScore() float64 {
	if a.PubDate.Valid {
		return float64(a.PubDate.Time.Unix())
	}

	return 0
}

type ArticleList []Article
