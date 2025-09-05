package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value []byte) error
}

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	logger *zap.SugaredLogger
}

func NewRedisCache(addr string, ttl time.Duration, logger *zap.SugaredLogger) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	// Проверим коннект один раз при создании
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warnw("redis not available", "addr", addr, "error", err)
	} else {
		logger.Infow("connected to redis", "addr", addr)
	}

	return &RedisCache{
		client: rdb,
		ttl:    ttl,
		logger: logger,
	}
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		r.logger.Warnw("redis get failed", "key", key, "error", err)
	}
	return val, err
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte) error {
	err := r.client.Set(ctx, key, value, r.ttl).Err()
	if err != nil {
		r.logger.Warnw("redis set failed", "key", key, "error", err)
	}
	return err
}
