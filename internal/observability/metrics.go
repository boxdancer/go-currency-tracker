package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics — минимальный интерфейс, который используют бизнес-пакеты.
type Metrics interface {
	// ObserveBackendCall отмечает длительность backend-вызова и успех/провал.
	ObserveBackendCall(d time.Duration, success bool)
	// CacheHit / CacheMiss — счётчики попаданий/промахов кэша.
	CacheHit()
	CacheMiss()
}

// Noop (для тестов)
type noopMetrics struct{}

func NewNoopMetrics() Metrics { return &noopMetrics{} }

func (n *noopMetrics) ObserveBackendCall(_ time.Duration, _ bool) {}
func (n *noopMetrics) CacheHit()                                {}
func (n *noopMetrics) CacheMiss()                               {}

// Prometheus реализация
type prometheusMetrics struct {
	backendLatency *prometheus.HistogramVec
	backendErrors  prometheus.Counter
	cacheHits      prometheus.Counter
	cacheMisses    prometheus.Counter
}

// NewPrometheusMetrics регистрирует и возвращает реализацию Metrics.
// Вызов функцию ровно один раз в main.
// Для тестов observability.NewNoopMetrics().
func NewPrometheusMetrics() Metrics {
	m := &prometheusMetrics{
		backendLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "client_backend_duration_seconds",
			Help:    "Duration of backend GetPrice calls in seconds, labeled by success",
			Buckets: prometheus.DefBuckets,
		}, []string{"success"}),
		backendErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "client_backend_errors_total",
			Help: "Number of backend errors",
		}),
		cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "cached_client_cache_hits_total",
			Help: "Number of cache hits served by CachedPriceClient",
		}),
		cacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "cached_client_cache_misses_total",
			Help: "Number of cache misses in CachedPriceClient",
		}),
	}

	// Регистрируем метрики (паника, если зарегистрировать дважды).
	prometheus.MustRegister(m.backendLatency, m.backendErrors, m.cacheHits, m.cacheMisses)

	return m
}

func (m *prometheusMetrics) ObserveBackendCall(d time.Duration, success bool) {
	label := "true"
	if !success {
		label = "false"
		m.backendErrors.Inc()
	}
	m.backendLatency.WithLabelValues(label).Observe(d.Seconds())
}

func (m *prometheusMetrics) CacheHit() {
	m.cacheHits.Inc()
}

func (m *prometheusMetrics) CacheMiss() {
	m.cacheMisses.Inc()
}
