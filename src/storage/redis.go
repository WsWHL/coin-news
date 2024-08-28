package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"news/src/config"
	"news/src/logger"
	"news/src/models"
	"strings"
)

const (
	NewsCategoryListKey = "news:category:%s"         // 指定分类文章列表
	NewsCategorySetKey  = "news:category:set:%s"     // 所有文章类别集合
	NewsOriginListKey   = "news:origin:%s"           // 指定网站文章列表
	NewsOriginSetKey    = "news:origin:set:%s"       // 指定网站文章集合
	NewsTokenKey        = "news:token:%s"            // 文章内容信息
	NewsOriginsListKey  = "news:origins:category:%s" // 类别来源网址列表
	NewsAllTokensKey    = "news:all:tokens"          // 所有文章 token 列表

	CoinSlugsListKey = "coin:slugs:%s"      // 指定币文章列表
	CoinNewsTokenKey = "coin:news:token:%s" // 币种文章信息
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

	article.Token = article.GenToken()
	// 保存文章信息
	key := fmt.Sprintf(NewsTokenKey, article.Token)
	if err := s.client.Set(ctx, key, article, 0).Err(); err != nil {
		return err
	}

	// 保存到分类列表
	key = fmt.Sprintf(NewsCategoryListKey, article.Category)
	if err := s.client.LPush(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存到分类集合
	key = fmt.Sprintf(NewsCategorySetKey, article.Category)
	if err := s.client.SAdd(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存到网站列表
	key = fmt.Sprintf(NewsOriginListKey, article.From)
	if err := s.client.LPush(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存到网站集合
	key = fmt.Sprintf(NewsOriginSetKey, article.From)
	if err := s.client.SAdd(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存类别来源列表
	key = fmt.Sprintf(NewsOriginsListKey, article.Category)
	if err := s.client.SAdd(ctx, key, article.From).Err(); err != nil {
		return err
	}

	// 保存所有文章 token 列表
	if err := s.client.LPush(ctx, NewsAllTokensKey, article.Token).Err(); err != nil {
		return err
	}

	return nil
}

func (s *RedisStorage) SaveCoin(article *models.Article) error {
	ctx := context.Background()

	article.Token = article.GenToken()
	// 保存币种文章列表
	key := fmt.Sprintf(CoinSlugsListKey, article.Category)
	if err := s.client.LPush(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存文章信息
	key = fmt.Sprintf(CoinNewsTokenKey, article.Token)
	if err := s.client.Set(ctx, key, article, 0).Err(); err != nil {
		return err
	}

	return nil
}

func (s *RedisStorage) GetHomeList(category string, page, size int) ([]*models.Article, int64, error) {
	var (
		articles []*models.Article
		count    int64
	)

	key := NewsAllTokensKey
	if category != "" {
		key = fmt.Sprintf(NewsCategoryListKey, category)
	}

	start := int64((page-1)*size) + 1
	stop := int64(page * size)
	tokens, err := s.client.LRange(context.Background(), key, start, stop).Result()
	if err != nil {
		return nil, 0, err
	}

	count = s.client.LLen(context.Background(), key).Val()
	for _, token := range tokens {
		if article, err := s.Get(token); err == nil {
			articles = append(articles, article)
		}
	}

	return articles, count, nil
}

func (s *RedisStorage) GetReadList(origins []string, category string) (map[string][]*models.Article, error) {
	keys := make([]string, 0, 10)
	for _, origin := range origins {
		keys = append(keys, fmt.Sprintf(NewsOriginSetKey, origin))
	}
	if category != "" {
		keys = append(keys, fmt.Sprintf(NewsCategorySetKey, category))
	}
	if len(keys) == 0 {
		return nil, nil
	}

	tokens, err := s.client.SInter(context.Background(), keys...).Result()
	if err != nil {
		return nil, err
	}

	articles := make(map[string][]*models.Article)
	for _, token := range tokens {
		if article, err := s.Get(token); err == nil {
			articles[article.From] = append(articles[article.From], article)
		}
	}

	return articles, nil
}

func (s *RedisStorage) GetListByCategory(category string) ([]*models.Article, error) {
	key := fmt.Sprintf(NewsCategoryListKey, category)
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

	key := fmt.Sprintf(NewsOriginListKey, origin)
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
	key := fmt.Sprintf(NewsOriginsListKey, category)
	origins, err := s.client.SMembers(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	return origins, nil
}

func (s *RedisStorage) NewsSearch(keyword string, page, size int) ([]*models.Article, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *RedisStorage) Restore() error {
	ctx := context.Background()

	keys := []string{
		NewsCategoryListKey,
		NewsCategorySetKey,
		NewsOriginListKey,
		NewsOriginSetKey,
		NewsTokenKey,
		NewsOriginsListKey,
		NewsAllTokensKey,
		CoinSlugsListKey,
		CoinNewsTokenKey,
	}

	for _, key := range keys {
		if strings.Contains(key, "%s") {
			key = fmt.Sprintf(key, "*")
		}
		keyList, err := s.client.Keys(ctx, key).Result()
		if err != nil {
			logger.Errorf("Error restoring key: %s, error: %v", key, err)
			continue
		}

		if len(keyList) > 0 {
			if err = s.client.Del(ctx, keyList...).Err(); err != nil {
				logger.Errorf("Error deleting key: %s, error: %v", key, err)
			}
		}
	}

	return nil
}
