package currency

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

type PriceClient interface {
	GetPrice(ctx context.Context, id, vs string) (float64, error)
}

type Service struct {
	client PriceClient
}

func NewService(c PriceClient) *Service {
	return &Service{client: c}
}

// GetMany получает цены для пар {id: vs} конкурентно.
// Пример входа: {"bitcoin":"usd", "ethereum":"usd"}.
func (s *Service) GetMany(ctx context.Context, pairs map[string]string) (map[string]map[string]float64, error) {
	results := make(map[string]map[string]float64)
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	for id, vs := range pairs {
		id, vs := id, vs // захват значений цикла
		g.Go(func() error {
			price, err := s.client.GetPrice(ctx, id, vs)
			if err != nil {
				return fmt.Errorf("%s->%s: %w", id, vs, err)
			}
			mu.Lock()
			if results[id] == nil {
				results[id] = make(map[string]float64)
			}
			results[id][vs] = price
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return results, err // частичные данные уже в results
	}
	return results, nil
}
