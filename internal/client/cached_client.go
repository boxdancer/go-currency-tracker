package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/boxdancer/go-currency-tracker/internal/cache"
	"github.com/boxdancer/go-currency-tracker/internal/observability"
	"github.com/boxdancer/go-currency-tracker/internal/price"
)

// CachedPriceClient оборачивает backend (любой price.PriceClient) и добавляет Redis-кэш.
type CachedPriceClient struct {
	backend price.PriceClient
	cache   cache.Cache
	metrics observability.Metrics
}

// NewCachedPriceClient принимает backend, реализацию cache.Cache и observability.Metrics.
// metrics может быть nil — тогда будет использован noop.
func NewCachedPriceClient(backend price.PriceClient, c cache.Cache, m observability.Metrics) *CachedPriceClient {
	if m == nil {
		m = observability.NewNoopMetrics()
	}
	return &CachedPriceClient{
		backend: backend,
		cache:   c,
		metrics: m,
	}
}

func (c *CachedPriceClient) GetPrice(ctx context.Context, id, vs string) (float64, error) {
	key := fmt.Sprintf("price:%s:%s", id, vs)

	// Попытка взять из кэша (best-effort)
	if c.cache != nil {
		if val, err := c.cache.Get(ctx, key); err == nil {
			var cached float64
			if unmarshalErr := json.Unmarshal([]byte(val), &cached); unmarshalErr == nil {
				c.metrics.CacheHit()
				return cached, nil
			}
			// если unmarshal не удался — продолжаем к backend
		}
		c.metrics.CacheMiss()
	}

	// В кэше нет — идём в backend
	start := time.Now()
	priceVal, err := c.backend.GetPrice(ctx, id, vs)
	c.metrics.ObserveBackendCall(time.Since(start), err == nil)
	if err != nil {
		return 0, err
	}

	// Сохраняем в кэш (ошибки от Set игнорируем)
	if c.cache != nil {
		if data, marshalErr := json.Marshal(priceVal); marshalErr == nil {
			_ = c.cache.Set(ctx, key, data)
		}
	}

	return priceVal, nil
}
