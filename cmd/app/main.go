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
		if _, err := fmt.Fprintln(w, "pong"); err != nil {
			log.Printf("Error writing ping response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
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
		
		if _, err := fmt.Fprintf(w, "BTC/USD: %.2f", price); err != nil {
			log.Printf("Error writing BTC price response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Конкурентное получение нескольких курсов
	http.HandleFunc("/rates", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		pairs := map[string]string{
			"bitcoin":  "usd",
			"ethereum": "usd",
			"usd":      "rub",
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
			if encodeErr := json.NewEncoder(w).Encode(map[string]any{
				"data":  data,
				"error": err.Error(),
			}); encodeErr != nil {
				log.Printf("Error encoding error response: %v", encodeErr)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Error encoding success response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	addr := ":8080"
	log.Printf("Server is running on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}