package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	cache *redis.Client
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
		_ = err
	}
	return nil, false
}

func (c *Cache) Set(ctx context.Context, key string, payload []byte, ttl time.Duration) {
	if c.cache == nil || ttl < 0 {
		return
	}
	_ = c.cache.Set(ctx, key, payload, ttl).Err()
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

func (c *Cache) SAdd(ctx context.Context, key string, members ...interface{}) error {
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

	payload, err := fn()
	if err != nil {
		return nil, err
	}

	c.Set(ctx, key, payload, ttl)
	return payload, nil
}