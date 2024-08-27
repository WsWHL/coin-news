package storage

import "news/src/models"

type ElasticsearchStorage struct {
}

func (s *ElasticsearchStorage) Save(article *models.Article) error {
	return nil
}
