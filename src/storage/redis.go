package storage

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"news/src/config"
	"news/src/models"
)

const (
	NewsCategoryKey        = "news:category:%s"         // 指定分类文章列表
	NewsOriginKey          = "news:origin:%s"           // 指定网站文章列表
	NewsTokenKey           = "news:token:%s"            // 文章内容信息
	NewsOriginsCategoryKey = "news:origins:category:%s" // 类别来源网址列表
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage() *RedisStorage {
	r := config.Cfg.Redis
	return &RedisStorage{
		client: redis.NewClient(&redis.Options{
			Addr:     r.Addr,
			Password: r.Password,
			DB:       r.DB,
		}),
	}
}

func (s *RedisStorage) Get(token string) (*models.Article, error) {
	article := &models.Article{}

	key := fmt.Sprintf(NewsTokenKey, token)
	if err := s.client.Get(context.Background(), key).Scan(article); err != nil {
		return nil, err
	}

	return article, nil
}

func (s *RedisStorage) Save(article *models.Article) error {
	ctx := context.Background()

	// 保存文章信息
	key := fmt.Sprintf(NewsTokenKey, article.Token)
	if err := s.client.Set(ctx, key, article, 0).Err(); err != nil {
		return err
	}

	// 保存到分类列表
	key = fmt.Sprintf(NewsCategoryKey, article.Category)
	if err := s.client.LPush(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存到网站列表
	key = fmt.Sprintf(NewsOriginKey, article.From)
	if err := s.client.LPush(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存类别来源列表
	key = fmt.Sprintf(NewsOriginsCategoryKey, article.Category)
	if err := s.client.SAdd(ctx, key, article.From).Err(); err != nil {
		return err
	}

	return nil
}

func (s *RedisStorage) GetListByCategory(category string) ([]*models.Article, error) {
	key := fmt.Sprintf(NewsCategoryKey, category)
	tokens, err := s.client.LRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	articles := make([]*models.Article, 0, len(tokens))
	for _, token := range tokens {
		if article, err := s.Get(token); err == nil {
			articles = append(articles, article)
		}
	}

	return articles, nil
}

func (s *RedisStorage) GetListByOrigin(origin string, page, size int) ([]*models.Article, int64, error) {
	start := int64((page-1)*size) + 1
	stop := int64(page * size)

	key := fmt.Sprintf(NewsOriginKey, origin)
	tokens, err := s.client.LRange(context.Background(), key, start, stop).Result()
	if err != nil {
		return nil, 0, err
	}
	count := s.client.LLen(context.Background(), key).Val()

	articles := make([]*models.Article, 0, len(tokens))
	for _, token := range tokens {
		if article, err := s.Get(token); err == nil {
			articles = append(articles, article)
		}
	}

	return articles, count, nil
}

func (s *RedisStorage) GetOriginsByCategory(category string) ([]string, error) {
	key := fmt.Sprintf(NewsOriginsCategoryKey, category)
	origins, err := s.client.SMembers(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	return origins, nil
}
