package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/boxdancer/go-currency-tracker/internal/cache"
	"github.com/boxdancer/go-currency-tracker/internal/client"
	"github.com/boxdancer/go-currency-tracker/internal/currency"
)

func main() {
	// Redis
	redisCache := cache.NewRedisCache("localhost:6379", time.Minute)

	// Базовый клиент CoinGecko
	cg := client.NewCoinGeckoClient(5 * time.Second)

	// Кэшированный клиент поверх cg
	cachedClient := client.NewCachedPriceClient(cg, redisCache)

	// Сервис использует cachedClient
	svc := currency.NewService(cachedClient)

	// /ping
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, "pong"); err != nil {
			log.Printf("Error writing ping response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// /btc-usd: теперь через cachedClient
	http.HandleFunc("/btc-usd", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		price, err := cachedClient.GetPrice(ctx, "bitcoin", "usd")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_, _ = fmt.Fprintf(w, "BTC/USD: %.2f", price)
	})

	// /rates: конкурентно через сервис
	http.HandleFunc("/rates", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		pairs := map[string]string{
			"bitcoin":  "usd",
			"ethereum": "usd",
			"usd":      "rub",
		}

		data, err := svc.GetMany(ctx, pairs)
		if err != nil {
			status := http.StatusPartialContent
			if len(data) == 0 {
				status = http.StatusBadGateway
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":  data,
				"error": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
	})

	addr := ":8080"
	log.Printf("Server is running on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
