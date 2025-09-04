package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(addr string, ttl time.Duration) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisCache{client: rdb, ttl: ttl}
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) Set(ctx context.Context, key, value string) error {
	return r.client.Set(ctx, key, value, r.ttl).Err()
}
