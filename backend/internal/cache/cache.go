package cache

import (
	"context"
	"log"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	cache *redis.Client
	sf    singleflight.Group
}

func New(cache *redis.Client) *Cache {
	return &Cache{cache: cache}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool) {
	if c.cache == nil {
		return nil, false
	}
	payload, err := c.cache.Get(ctx, key).Bytes()
	if err == nil {
		return payload, true
	}
	if err != redis.Nil {
		log.Printf("cache: redis error for key %q: %v", key, err)
	}
	return nil, false
}

func (c *Cache) Set(ctx context.Context, key string, payload []byte, ttl time.Duration) {
	if c.cache == nil || ttl < 0 {
		return
	}
	if err := c.cache.Set(ctx, key, payload, ttl).Err(); err != nil {
		log.Printf("cache: redis set error for key %q: %v", key, err)
	}
}

func (c *Cache) Del(ctx context.Context, key string) error {
	if c.cache == nil {
		return nil
	}
	return c.cache.Del(ctx, key).Err()
}

func (c *Cache) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanIterator {
	if c.cache == nil {
		return nil
	}
	return c.cache.Scan(ctx, cursor, match, count).Iterator()
}

func (c *Cache) SAdd(ctx context.Context, key string, members ...any) error {
	if c.cache == nil {
		return nil
	}
	return c.cache.SAdd(ctx, key, members...).Err()
}

func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	if c.cache == nil {
		return nil, nil
	}
	return c.cache.SMembers(ctx, key).Result()
}

func (c *Cache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() ([]byte, error)) ([]byte, error) {
	if cached, ok := c.Get(ctx, key); ok {
		return cached, nil
	}

	result, err, _ := c.sf.Do(key, func() (any, error) {
		// Re-check cache inside the singleflight to avoid a stampede after the
		// first waiter already populated it.
		if cached, ok := c.Get(ctx, key); ok {
			return cached, nil
		}
		payload, err := fn()
		if err != nil {
			return nil, err
		}
		c.Set(ctx, key, payload, ttl)
		return payload, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]byte), nil
}
