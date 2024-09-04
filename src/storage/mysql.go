package storage

import (
	"errors"
	"gorm.io/gorm"
	"news/src/models"
	"time"
)

type MySQLStorage struct {
	DB *gorm.DB
}

func NewMySQLStorage(version int64) *MySQLStorage {
	return &MySQLStorage{
		DB: models.DB,
	}
}

func (s *MySQLStorage) GetVersion() (int64, error) {
	return 0, errors.New("not implemented")
}

func (s *MySQLStorage) SetVersion(version int64) {
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

func (s *MySQLStorage) SaveCoin(article *models.Article) error {
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

func (s *MySQLStorage) GetHomeList(category string, page, size int) ([]*models.Article, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *MySQLStorage) GetReadList(origin []string, category string) (map[string][]*models.Article, error) {
	return nil, errors.New("not implemented")
}

func (s *MySQLStorage) GetListByCategory(category string) ([]*models.Article, error) {
	return nil, errors.New("not implemented")
}

func (s *MySQLStorage) GetListByOrigin(origin string, page, size int) ([]*models.Article, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *MySQLStorage) GetOriginsByCategory(category string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (s *MySQLStorage) NewsSearch(keyword string, page, size int) ([]*models.Article, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *MySQLStorage) Restore() error {
	return nil
}
