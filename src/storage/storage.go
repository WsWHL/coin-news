package storage

import (
	"errors"
	"news/src/models"
)

type Strategy interface {
	Get(token string) (*models.Article, error)
	Save(article *models.Article) error
	SaveCoin(article *models.Article) error
	GetHomeList(category string, page, size int) ([]*models.Article, int64, error)
	GetReadList(origin []string, category string) (map[string][]*models.Article, error)
	GetListByCategory(category string) ([]*models.Article, error)
	GetListByOrigin(origin string, page, size int) ([]*models.Article, int64, error)
	GetOriginsByCategory(category string) ([]string, error)
	NewsSearch(keyword string, page, size int) ([]*models.Article, int64, error)
	Restore() error
}

type Service struct {
	storages []Strategy
}

func NewService() *Service {
	return &Service{
		storages: []Strategy{
			NewMySQLStorage(),
			NewRedisStorage(),
			NewElasticsearchStorage(),
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
		if e := store.Save(article); e != nil {
			err = errors.Join(err, e)
		}

		return false // 每个存储都需要保存
	})

	return err
}

func (s *Service) SaveCoin(article *models.Article) error {
	var err error
	s.fetch(func(store Strategy) bool {
		_ = store.SaveCoin(article)
		return false // 每个存储都需要保存
	})

	return err
}

func (s *Service) GetHomeList(category string, page, size int) ([]*models.Article, int64) {
	var (
		articles []*models.Article
		count    int64
		err      error
	)
	s.fetch(func(store Strategy) bool {
		if articles, count, err = store.GetHomeList(category, page, size); err != nil {
			return false
		}

		return true
	})

	return articles, count
}

func (s *Service) GetReadList(origins []string, category string) (map[string][]*models.Article, error) {
	var (
		readList map[string][]*models.Article
		err      error
	)
	s.fetch(func(store Strategy) bool {
		if readList, err = store.GetReadList(origins, category); err != nil {
			return false
		}

		return true
	})

	return readList, err
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

func (s *Service) NewsSearch(keyword string, page, size int) ([]*models.Article, int64) {
	var (
		articles []*models.Article
		count    int64
		err      error
	)
	s.fetch(func(store Strategy) bool {
		if articles, count, err = store.NewsSearch(keyword, page, size); err != nil {
			return false
		}

		return true
	})

	return articles, count
}

func (s *Service) Restore() error {
	var err error
	s.fetch(func(store Strategy) bool {
		if e := store.Restore(); e != nil {
			err = errors.Join(err, e)
		}
		return false // 每个存储都需要恢复
	})

	return err
}
