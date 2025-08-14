package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// структура для парсинга ответа
type coinGeckoResponse struct {
	Bitcoin struct {
		USD float64 `json:"usd"`
	} `json:"bitcoin"`
}

// GetBTCUSD запрашивает курс BTC в USD с CoinGecko
func GetBTCUSD() (float64, error) {
	url := "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd"

	// HTTP-клиент с таймаутом
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("ошибка при запросе API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("получен некорректный статус: %d", resp.StatusCode)
	}

	var data coinGeckoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("ошибка при парсинге JSON: %w", err)
	}

	return data.Bitcoin.USD, nil
}
