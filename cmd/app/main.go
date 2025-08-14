package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/boxdancer/go-currency-tracker/internal/client"
)

func main() {
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "pong")
	})

	http.HandleFunc("/btc-usd", func(w http.ResponseWriter, r *http.Request) {
		rate, err := client.GetBTCUSD()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "BTC/USD: %.2f", rate)
	})

	addr := ":8080"
	log.Printf("Server is running on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
