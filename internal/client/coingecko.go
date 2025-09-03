package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CoinGeckoClient struct {
	http    *http.Client
	baseURL string
}

func NewCoinGeckoClient(timeout time.Duration) *CoinGeckoClient {
	return &CoinGeckoClient{
		http: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.coingecko.com",
	}
}

// GetPrice возвращает цену монеты id в фиате vs (например: id="bitcoin", vs="usd").
func (c *CoinGeckoClient) GetPrice(ctx context.Context, id, vs string) (float64, error) {
	url := fmt.Sprintf("%s/api/v3/simple/price?ids=%s&vs_currencies=%s", c.baseURL, id, vs)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Логируем ошибку закрытия, но не прерываем выполнение
			// так как основная операция уже завершена
			fmt.Printf("Warning: error closing response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode json: %w", err)
	}

	priceMap, ok := data[id]
	if !ok {
		return 0, fmt.Errorf("no id %q in response", id)
	}
	price, ok := priceMap[vs]
	if !ok {
		return 0, fmt.Errorf("no vs %q for id %q in response", vs, id)
	}
	return price, nil
}
