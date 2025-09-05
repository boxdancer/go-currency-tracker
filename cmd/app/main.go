package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // регистрирует pprof handlers на DefaultServeMux
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/boxdancer/go-currency-tracker/internal/cache"
	"github.com/boxdancer/go-currency-tracker/internal/client"
	"github.com/boxdancer/go-currency-tracker/internal/currency"
	"github.com/boxdancer/go-currency-tracker/internal/observability"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed, labeled by method, path and status",
		},
		[]string{"method", "path", "status"},
	)

	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds, labeled by method and path",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	// Регистрируем метрики в дефолтном реестре
	prometheus.MustRegister(httpRequests, httpDuration)
}

// statusRecorder — вспомогательный writer, чтобы сохранить статус-код ответа
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func instrumentHandler(path string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		h(rec, r)
		duration := time.Since(start).Seconds()
		httpDuration.WithLabelValues(r.Method, path).Observe(duration)
		httpRequests.WithLabelValues(r.Method, path, strconv.Itoa(rec.status)).Inc()
	}
}

func main() {
	// logger
	logger, _ := zap.NewDevelopment()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Printf("logger.Sync() error: %v\n", err)
		}
	}()
	sugar := logger.Sugar()

	// Observability:
	metrics := observability.NewPrometheusMetrics()

	// Загружаем .env
	_ = godotenv.Load()

	redisAddr := os.Getenv("REDIS_ADDR")

	// Redis cache -> CoinGecko client -> cached client -> service
	redisCache := cache.NewRedisCache(redisAddr, time.Minute, sugar)
	cg := client.NewCoinGeckoClient(5 * time.Second)
	cachedClient := client.NewCachedPriceClient(cg, redisCache, metrics)
	svc := currency.NewService(cachedClient)

	// handlers
	http.HandleFunc("/ping", instrumentHandler("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "pong")
	}))

	http.HandleFunc("/btc-usd", instrumentHandler("/btc-usd", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		price, err := cachedClient.GetPrice(ctx, "bitcoin", "usd")
		if err != nil {
			sugar.Errorf("btc-usd: error getting price: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_, _ = fmt.Fprintf(w, "BTC/USD: %.2f", price)
	}))

	http.HandleFunc("/rates", instrumentHandler("/rates", func(w http.ResponseWriter, r *http.Request) {
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
	}))

	// Prometheus metrics endpoint (scrape target)
	http.Handle("/metrics", promhttp.Handler())

	// NOTE: pprof handlers are available at /debug/pprof/* because of `_ "net/http/pprof"`

	addr := ":8080"
	srv := &http.Server{
		Addr:    addr,
		Handler: nil, // DefaultServeMux (мы использовали http.Handle/HandleFunc)
	}

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		sugar.Infof("starting server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done() // ждём сигнал
	sugar.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		sugar.Errorf("server shutdown error: %v", err)
	} else {
		sugar.Info("server stopped gracefully")
	}
}
