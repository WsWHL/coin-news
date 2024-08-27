package storage

import (
	"gorm.io/gorm"
	"news/src/models"
	"time"
)

type MySQLStorage struct {
	DB *gorm.DB
}

func NewMySQLStorage() *MySQLStorage {
	return &MySQLStorage{
		DB: models.DB,
	}
}

func (s *MySQLStorage) Get(token string) (*models.Article, error) {
	article := &models.Article{}
	err := s.DB.Model(article).Where("token = ?", token).First(article).Error

	return article, err
}

func (s *MySQLStorage) Save(article *models.Article) error {
	existingArticle := &models.Article{}
	article.Token = article.GenToken()
	s.DB.Model(existingArticle).Where("title = ?", article.Title).First(existingArticle)
	if existingArticle.ID > 0 {
		article.ID = existingArticle.ID
		article.CreateTime = existingArticle.CreateTime
	} else {
		article.CreateTime = time.Now()
	}

	article.UpdateTime = time.Now()
	return s.DB.Save(article).Error
}

func (s *MySQLStorage) GetListByCategory(category string) ([]*models.Article, error) {
	return nil, nil
}

func (s *MySQLStorage) GetListByOrigin(origin string, page, size int) ([]*models.Article, int64, error) {
	return nil, 0, nil
}

func (s *MySQLStorage) GetOriginsByCategory(category string) ([]string, error) {
	return nil, nil
}
