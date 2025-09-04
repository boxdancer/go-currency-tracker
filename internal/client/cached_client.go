package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/boxdancer/go-currency-tracker/internal/price"
	"github.com/redis/go-redis/v9"
)

// CachedPriceClient оборачивает backend (любой price.PriceClient) и добавляет Redis-кэш.
type CachedPriceClient struct {
	backend price.PriceClient
	redis   *redis.Client
	ttl     time.Duration
}

func NewCachedPriceClient(backend price.PriceClient, rdb *redis.Client, ttl time.Duration) *CachedPriceClient {
	return &CachedPriceClient{
		backend: backend,
		redis:   rdb,
		ttl:     ttl,
	}
}

func (c *CachedPriceClient) GetPrice(ctx context.Context, id, vs string) (float64, error) {
	key := fmt.Sprintf("price:%s:%s", id, vs)

	// Попытка взять из кэша (best-effort)
	if c.redis != nil {
		if val, err := c.redis.Get(ctx, key).Result(); err == nil {
			var cached float64
			if unmarshalErr := json.Unmarshal([]byte(val), &cached); unmarshalErr == nil {
				return cached, nil
			}
			// если unmarshal не удался — продолжаем к backend
		}
	}

	// В кэше нет — идём в backend
	priceVal, err := c.backend.GetPrice(ctx, id, vs)
	if err != nil {
		return 0, err
	}

	// Сохраняем в кэш (ошибки от Set игнорируем)
	if c.redis != nil {
		if data, marshalErr := json.Marshal(priceVal); marshalErr == nil {
			_ = c.redis.Set(ctx, key, data, c.ttl).Err()
		}
	}

	return priceVal, nil
}
