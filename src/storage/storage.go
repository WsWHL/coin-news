package storage

import (
	"news/src/models"
)

type Strategy interface {
	Get(token string) (*models.Article, error)
	Save(article *models.Article) error
	GetListByCategory(category string) ([]*models.Article, error)
	GetListByOrigin(origin string, page, size int) ([]*models.Article, int64, error)
	GetOriginsByCategory(category string) ([]string, error)
}

type Service struct {
	storages []Strategy
}

func NewService() *Service {
	return &Service{
		storages: []Strategy{
			NewMySQLStorage(),
			NewRedisStorage(),
		},
	}
}

func NewServiceCacheFirst() *Service {
	return &Service{
		storages: []Strategy{
			NewRedisStorage(),
			NewMySQLStorage(),
		},
	}
}

func (s *Service) fetch(f func(Strategy) bool) {
	for _, storage := range s.storages {
		if f(storage) {
			return
		}
	}
}

func (s *Service) Get(token string) (*models.Article, error) {
	var (
		article *models.Article
		err     error
	)
	s.fetch(func(store Strategy) bool {
		if article, err = store.Get(token); err != nil {
			return false
		}

		return true
	})

	return article, err
}

func (s *Service) Save(article *models.Article) error {
	var err error
	s.fetch(func(store Strategy) bool {
		if err = store.Save(article); err != nil {
			return false
		}

		return true
	})

	return err
}

func (s *Service) GetListByCategory(category string) ([]*models.Article, error) {
	var (
		articles []*models.Article
		err      error
	)
	s.fetch(func(store Strategy) bool {
		if articles, err = store.GetListByCategory(category); err != nil {
			return false
		}

		return true
	})

	return articles, err
}

func (s *Service) GetListByOrigin(origin string, page, size int) ([]*models.Article, int64, error) {
	var (
		articles []*models.Article
		count    int64
		err      error
	)
	s.fetch(func(store Strategy) bool {
		if articles, count, err = store.GetListByOrigin(origin, page, size); err != nil {
			return false
		}

		return true
	})

	return articles, count, err
}

func (s *Service) GetOriginsByCategory(category string) ([]string, error) {
	var (
		origins []string
		err     error
	)
	s.fetch(func(store Strategy) bool {
		if origins, err = store.GetOriginsByCategory(category); err != nil {
			return false
		}

		return true
	})

	return origins, err
}
