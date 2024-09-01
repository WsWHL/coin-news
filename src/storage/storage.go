package storage

import (
	"errors"
	"news/src/logger"
	"news/src/models"
)

var (
	versionChan chan int64
)

type Strategy interface {
	GetVersion() (int64, error)
	SetVersion(version int64)
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

	quit chan struct{}
}

func init() {
	versionChan = make(chan int64, 1)
}

func NewService() *Service {
	r := NewRedisStorage(0)
	version, _ := r.GetVersion()
	s := NewServiceWithVersion(version)

	// 监听版本变化
	go listenVersion(s)

	return s
}

func NewServiceWithVersion(version int64) *Service {
	logger.Infof("Initializing service with data version: %d", version)
	return &Service{
		storages: []Strategy{
			NewMySQLStorage(version),
			NewRedisStorage(version),
			NewElasticsearchStorage(version),
		},
		quit: make(chan struct{}, 1),
	}
}

func listenVersion(s *Service) {
	for {
		select {
		case version := <-versionChan:
			for _, store := range s.storages {
				store.SetVersion(version)
			}
			logger.Infof("Received version change: %d", version)
		case <-s.quit:
			logger.Infof("Service stopped.")
			return
		}
	}
}

func NotifyVersion(version int64) {
	versionChan <- version
}

func (s *Service) SetVersion(version int64) {
	for _, store := range s.storages {
		store.SetVersion(version)
	}
}

func (s *Service) Release() {
	close(s.quit)
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
	var errs []error
	s.fetch(func(store Strategy) bool {
		if err := store.Save(article); err != nil {
			errs = append(errs, err)
		}

		return false // 每个存储都需要保存
	})

	return errors.Join(errs...)
}

func (s *Service) SaveCoin(article *models.Article) error {
	var errs []error
	s.fetch(func(store Strategy) bool {
		if err := store.SaveCoin(article); err != nil {
			errs = append(errs, err)
		}
		return false
	})

	return errors.Join(errs...)
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
	var errs []error
	s.fetch(func(store Strategy) bool {
		if err := store.Restore(); err != nil {
			errs = append(errs, err)
		}
		return false // 每个存储都需要恢复
	})

	return errors.Join(errs...)
}
