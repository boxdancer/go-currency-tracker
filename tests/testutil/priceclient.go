package testutil

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Key для Responses/Errors
type Key struct{ ID, VS string }

type FakePriceClient struct {
	Responses map[Key]float64
	Errors    map[Key]error
	Delay     time.Duration // опциональная задержка для имитации сети
}

func (f *FakePriceClient) GetPrice(ctx context.Context, id, vs string) (float64, error) {
	if f.Delay > 0 {
		select {
		case <-time.After(f.Delay):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	k := Key{ID: id, VS: vs}
	if err, ok := f.Errors[k]; ok {
		return 0, err
	}
	if v, ok := f.Responses[k]; ok {
		return v, nil
	}
	return 0, fmt.Errorf("no mock for %s:%s", id, vs)
}

// Утилита для быстрого создания ошибок
func Err(msg string) error { return errors.New(msg) }
