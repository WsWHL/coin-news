package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"news/src/config"
	"news/src/logger"
	"news/src/models"
	"strconv"
	"strings"
)

const (
	NewsCategoryZSetKey   = "news:category:%s"         // 所有文章类别集合
	NewsOriginZSetKey     = "news:origin:%s"           // 指定网站文章集合
	NewsTokenKey          = "news:tokens:%s"           // 文章内容信息
	NewsOriginsSetKey     = "news:origins:category:%s" // 类别来源网址列表
	NewsAllTokensZSetKey  = "news:all:tokens"          // 所有文章 token 列表
	TempNewsOriginZSetKey = "temp:news:origin:%s"      // 临时多网站合集

	CoinSlugsListKey = "coin:slugs:%s"      // 指定币文章列表
	CoinNewsTokenKey = "coin:news:token:%s" // 币种文章信息

	DataVersionZSetKye = "data:versions" // 数据版本列表
)

type RedisStorage struct {
	client *redis.Client

	dataVersion int64
}

func NewRedisStorage(version int64) *RedisStorage {
	r := config.Cfg.Redis
	return &RedisStorage{
		client: redis.NewClient(&redis.Options{
			Addr:     r.Addr,
			Password: r.Password,
			DB:       r.DB,
		}),
		dataVersion: version,
	}
}

func (s *RedisStorage) sKey(key string, args ...interface{}) string {
	if s.dataVersion > 0 {
		key = fmt.Sprintf("%d:%s", s.dataVersion, key)
	}

	return fmt.Sprintf(key, args...)
}

func (s *RedisStorage) GetVersion() (int64, error) {
	// get data version from redis
	ctx := context.Background()
	versions, err := s.client.ZRevRange(ctx, DataVersionZSetKye, 0, 0).Result()
	if err != nil {
		return 0, err
	}

	if len(versions) > 0 {
		v, _ := strconv.Atoi(versions[0])
		s.dataVersion = int64(v)
	}

	return s.dataVersion, nil
}

func (s *RedisStorage) SetVersion(version int64) {
	err := s.client.ZAdd(context.Background(), DataVersionZSetKye, redis.Z{
		Member: strconv.Itoa(int(version)),
		Score:  float64(version),
	}).Err()
	if err != nil {
		logger.Errorf("set data version error: %v", err)
	}

	s.dataVersion = version
}

func (s *RedisStorage) Get(token string) (*models.Article, error) {
	article := &models.Article{}

	key := s.sKey(NewsTokenKey, token)
	if err := s.client.Get(context.Background(), key).Scan(article); err != nil {
		return nil, err
	}

	return article, nil
}

func (s *RedisStorage) Save(article *models.Article) error {
	ctx := context.Background()

	article.Token = article.GenToken()
	score := article.GetScore()
	// 保存文章信息
	key := s.sKey(NewsTokenKey, article.Token)
	if err := s.client.Set(ctx, key, article, 0).Err(); err != nil {
		return err
	}

	// 保存到分类集合
	key = s.sKey(NewsCategoryZSetKey, article.Category)
	if err := s.client.ZAdd(ctx, key, redis.Z{
		Member: article.Token,
		Score:  score,
	}).Err(); err != nil {
		return err
	}

	// 保存到网站集合
	key = s.sKey(NewsOriginZSetKey, article.From)
	if err := s.client.ZAdd(ctx, key, redis.Z{
		Member: article.Token,
		Score:  score,
	}).Err(); err != nil {
		return err
	}

	// 保存类别来源列表
	key = s.sKey(NewsOriginsSetKey, article.Category)
	if err := s.client.SAdd(ctx, key, article.From).Err(); err != nil {
		return err
	}

	// 保存所有文章 token 列表
	key = s.sKey(NewsAllTokensZSetKey)
	if err := s.client.ZAdd(ctx, key, redis.Z{
		Member: article.Token,
		Score:  score,
	}).Err(); err != nil {
		return err
	}

	return nil
}

func (s *RedisStorage) SaveCoin(article *models.Article) error {
	ctx := context.Background()

	article.Token = article.GenToken()
	// 保存币种文章列表
	key := s.sKey(CoinSlugsListKey, article.Category)
	if err := s.client.LPush(ctx, key, article.Token).Err(); err != nil {
		return err
	}

	// 保存文章信息
	key = s.sKey(CoinNewsTokenKey, article.Token)
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

	key := s.sKey(NewsAllTokensZSetKey)
	if category != "" {
		key = s.sKey(NewsCategoryZSetKey, category)
	}

	start := int64((page - 1) * size)
	stop := int64(page*size) - 1
	tokens, err := s.client.ZRevRange(context.Background(), key, start, stop).Result()
	if err != nil {
		return nil, 0, err
	}

	count = s.client.ZCard(context.Background(), key).Val()
	for _, token := range tokens {
		if article, err := s.Get(token); err == nil {
			articles = append(articles, article)
		}
	}

	return articles, count, nil
}

func (s *RedisStorage) GetReadList(origins []string, category string) (map[string][]*models.Article, error) {
	tempUnionKey := fmt.Sprintf(TempNewsOriginZSetKey, strings.Join(origins, "."))
	keys := make([]string, 0, 10)
	if len(origins) == 1 {
		keys = append(keys, s.sKey(NewsOriginZSetKey, origins[0]))
	} else {
		unionKeys := make([]string, 0, len(origins))
		for _, origin := range origins {
			unionKeys = append(unionKeys, s.sKey(NewsOriginZSetKey, origin))
		}
		err := s.client.ZUnionStore(context.Background(), tempUnionKey, &redis.ZStore{
			Keys: unionKeys,
		}).Err()
		if err != nil {
			return nil, err
		}
		keys = append(keys, tempUnionKey)
	}
	if category != "" {
		keys = append(keys, s.sKey(NewsCategoryZSetKey, category))
	}
	if len(keys) == 0 {
		return nil, nil
	}

	tokens, err := s.client.ZInter(context.Background(), &redis.ZStore{
		Keys: keys,
	}).Result()
	if err != nil {
		return nil, err
	}

	articles := make(map[string][]*models.Article)
	for _, token := range tokens {
		if article, err := s.Get(token); err == nil {
			articles[article.From] = append(articles[article.From], article)
		}
	}

	// delete temp union key
	if len(origins) > 0 {
		s.client.Del(context.Background(), tempUnionKey)
	}

	return articles, nil
}

func (s *RedisStorage) GetListByCategory(category string) ([]*models.Article, error) {
	key := s.sKey(NewsCategoryZSetKey, category)
	tokens, err := s.client.ZRange(context.Background(), key, 0, -1).Result()
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
	start := int64((page - 1) * size)
	stop := int64(page*size) - 1

	key := s.sKey(NewsOriginZSetKey, origin)
	tokens, err := s.client.ZRevRange(context.Background(), key, start, stop).Result()
	if err != nil {
		return nil, 0, err
	}

	count := s.client.ZCard(context.Background(), key).Val()
	articles := make([]*models.Article, 0, len(tokens))
	for _, token := range tokens {
		if article, err := s.Get(token); err == nil {
			articles = append(articles, article)
		}
	}

	return articles, count, nil
}

func (s *RedisStorage) GetOriginsByCategory(category string) ([]string, error) {
	key := s.sKey(NewsOriginsSetKey, category)
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

	versions, err := s.client.ZRevRange(ctx, DataVersionZSetKye, 3, -1).Result()
	if err != nil {
		logger.Errorf("Error getting data version: %v", err)
		return err
	}

	for _, version := range versions {
		key := fmt.Sprintf("%s:*", version)
		keys, err := s.client.Keys(ctx, key).Result()
		if err != nil {
			logger.Errorf("Error restoring key: %s, error: %v", key, err)
			continue
		}

		if len(keys) > 0 {
			if err = s.client.Del(ctx, keys...).Err(); err != nil {
				logger.Errorf("Error deleting key: %s, error: %v", key, err)
			}
		}
		s.client.ZRem(ctx, DataVersionZSetKye, version)
	}

	return nil
}
