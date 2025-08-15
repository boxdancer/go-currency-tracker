package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/boxdancer/go-currency-tracker/internal/client"
	"github.com/boxdancer/go-currency-tracker/internal/currency"
)

func main() {
	cg := client.NewCoinGeckoClient(5 * time.Second)
	svc := currency.NewService(cg)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})

	// Простой эндпоинт для BTC/USD (демо)
	http.HandleFunc("/btc-usd", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		price, err := cg.GetPrice(ctx, "bitcoin", "usd")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		fmt.Fprintf(w, "BTC/USD: %.2f", price)
	})

	// Конкурентное получение нескольких курсов
	http.HandleFunc("/rates", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		pairs := map[string]string{
			"bitcoin":  "usd",
			"ethereum": "usd",
			"usd": "rub",
			// Можно добавить "bitcoin":"eur" и т.п.
			// Фиат->фиат добавим позже через другой провайдер.
		}

		data, err := svc.GetMany(ctx, pairs)
		if err != nil {
			// Вернём частичные данные плюс 206 или 502; выберем 206
			// если что-то частично удалось.
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
