package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"net/http"
	"news/src/config"
	"news/src/logger"
	"news/src/models"
	"os"
	"sort"
	"strings"
)

const (
	IndexMapping = `
{
    "mappings": {
        "properties": {
            "id": {
                "type": "integer"
            },
            "token": {
                "type": "keyword"
            },
            "title": {
                "type": "text",
                "analyzer": "autocomplete",
                "search_analyzer": "standard"
            },
            "title_ch": {
                "type": "text"
            },
            "category": {
                "type": "keyword"
            },
            "from": {
                "type": "keyword"
            },
            "author": {
                "type": "keyword"
            },
            "abstract": {
                "type": "text"
            },
            "image": {
                "type": "text"
            },
            "link": {
                "type": "text"
            },
            "pub_date": {
                "properties": {
                    "Valid": {
                        "type": "boolean"
                    },
                    "Time": {
                        "type": "date"
                    }
                }
            },
            "reads": {
                "type": "integer"
            },
            "interactions": {
                "type": "integer"
            },
            "comments": {
                "type": "integer"
            },
            "notes": {
                "type": "text"
            },
            "create_time": {
                "type": "date"
            },
            "update_time": {
                "type": "date"
            }
        }
    },
    "settings": {
        "analysis": {
            "filter": {
                "autocomplete_filter": {
                    "type": "edge_ngram",
                    "min_gram": 1,
                    "max_gram": 20
                }
            },
            "analyzer": {
                "autocomplete": {
                    "type": "custom",
                    "tokenizer": "standard",
                    "filter": [
                        "lowercase",
                        "autocomplete_filter"
                    ]
                }
            }
        }
    },
    "aliases": {}
}
`
)

type ElasticsearchStorage struct {
	client *elasticsearch.Client
	index  string
}

func NewElasticsearchStorage(version int64) *ElasticsearchStorage {
	log := &elastictransport.JSONLogger{
		Output: os.Stdout,
	}
	if config.Cfg.API.Mode == "debug" {
		log.EnableRequestBody = true
		log.EnableResponseBody = true
	}

	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{config.Cfg.Elastic.Addr},
		Username:  config.Cfg.Elastic.Username,
		Password:  config.Cfg.Elastic.Password,
		Logger:    log,
	})
	if err != nil {
		panic(err)
	}

	elastic := &ElasticsearchStorage{
		client: client,
		index:  config.Cfg.Elastic.Index,
	}
	if version > 0 {
		elastic.index = fmt.Sprintf("%s.%d", config.Cfg.Elastic.Index, version)
	}

	info, err := elastic.client.Info()
	if err != nil {
		logger.Errorf("Error getting Elasticsearch info: %v", err)
		return nil
	}
	logger.Debugf("Elasticsearch cluster info: %s", info)

	// init index if not exists
	_ = elastic.init()

	return elastic
}

func (s *ElasticsearchStorage) GetVersion() (int64, error) {
	return 0, errors.New("not implemented")
}

func (s *ElasticsearchStorage) SetVersion(version int64) {
	if version > 0 {
		s.index = fmt.Sprintf("%s.%d", config.Cfg.Elastic.Index, version)
	}
}

func (s *ElasticsearchStorage) Get(token string) (*models.Article, error) {
	return nil, errors.New("not implemented")
}

func (s *ElasticsearchStorage) Save(article *models.Article) error {
	article.Token = article.GenToken()
	resp, err := s.client.Create(s.index, article.Token, bytes.NewReader(article.Bytes()))
	if err != nil {
		logger.Errorf("Error saving article to Elasticsearch: %v", err)
		return err
	}

	if resp.StatusCode == http.StatusConflict {
		body, _ := json.Marshal(map[string]interface{}{
			"doc": article,
		})
		resp, err = s.client.Update(s.index, article.Token, bytes.NewReader(body))
		if err != nil {
			logger.Errorf("Error updating article in Elasticsearch: %v", err)
			return err
		}
	}

	logger.Infof("Saved article to Elasticsearch: %s", resp)
	return nil
}

func (s *ElasticsearchStorage) SaveCoin(article *models.Article) error {
	return nil
}

func (s *ElasticsearchStorage) GetHomeList(category string, page, size int) ([]*models.Article, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *ElasticsearchStorage) GetReadList(origins []string, category string) (map[string][]*models.Article, error) {
	return nil, errors.New("not implemented")
}

func (s *ElasticsearchStorage) GetListByCategory(category string) ([]*models.Article, error) {
	return nil, errors.New("not implemented")
}

func (s *ElasticsearchStorage) GetListByOrigin(origin string, page, size int) ([]*models.Article, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *ElasticsearchStorage) GetOriginsByCategory(category string) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (s *ElasticsearchStorage) NewsSearch(keyword string, page, size int) ([]*models.Article, int64, error) {
	var (
		articles []*models.Article
		count    int64
	)

	resp, err := s.client.Search(
		s.client.Search.WithIndex(s.index),
		s.client.Search.WithQuery(keyword),
		s.client.Search.WithFrom((page-1)*size),
		s.client.Search.WithSize(size),
		s.client.Search.WithSort("pub_date.Time:desc", "reads:desc"),
		s.client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		logger.Errorf("Error searching articles in Elasticsearch: %v", err)
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body := search.Response{}
		if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
			logger.Errorf("Error decoding search result: %v", err)
			return nil, 0, err
		}

		for _, hit := range body.Hits.Hits {
			var article models.Article
			if err = json.Unmarshal(hit.Source_, &article); err != nil {
				logger.Errorf("Error unmarshalling article: %v", err)
				continue
			}
			articles = append(articles, &article)
		}
		count = body.Hits.Total.Value
	}

	return articles, count, nil
}

func (s *ElasticsearchStorage) Restore() error {
	mixIndex := fmt.Sprintf("%s.*", config.Cfg.Elastic.Index)
	resp, err := s.client.Cat.Indices(s.client.Cat.Indices.WithIndex(mixIndex), s.client.Cat.Indices.WithFormat("JSON"))
	if err != nil {
		logger.Errorf("Error getting indices: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body := make([]struct {
		Index string `json:"index"`
	}, 0)
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		logger.Errorf("Error unmarshalling index list: %v", err)
		return err
	}

	if len(body) > 3 {
		indexes := make([]string, 0, len(body))
		for _, i := range body {
			indexes = append(indexes, i.Index)
		}
		sort.Strings(indexes)
		indexes = indexes[:len(indexes)-3] // keep last 3 versions

		resp, err = s.client.Indices.Delete(indexes)
		if err != nil {
			logger.Errorf("Error deleting old indices: %v", err)
			return err
		}
	}

	return nil
}

// InitIndex initializes Elasticsearch index
func (s *ElasticsearchStorage) init() error {
	resp, err := s.client.Indices.Exists([]string{s.index})
	if err != nil {
		logger.Errorf("Error checking index existence: %v", err)
		return err
	}
	if resp.StatusCode != http.StatusNotFound {
		logger.Infof("Index %s already exists", s.index)
		return nil
	}

	reader := strings.NewReader(IndexMapping)
	resp, err = s.client.Indices.Create(s.index, s.client.Indices.Create.WithBody(reader))
	if err != nil {
		logger.Errorf("Error creating index: %v", err)
		return err
	}

	if resp.IsError() {
		logger.Errorf("Error creating index. resp: %s", resp)
	}

	return nil
}
