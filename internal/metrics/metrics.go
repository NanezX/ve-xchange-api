package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// ProviderFetchTotal counts every fetch attempt per provider.
	// status label is "success" or "failure".
	ProviderFetchTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "provider_fetch_total",
		Help: "Total number of provider fetch attempts by status.",
	}, []string{"provider", "status"})

	// ProviderConsecutiveFailures tracks the current failure streak per provider.
	// Resets to 0 after a successful fetch.
	ProviderConsecutiveFailures = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "provider_consecutive_failures",
		Help: "Current consecutive failure streak per provider.",
	}, []string{"provider"})

	// RateValue exposes the latest exchange rate as a gauge, suitable for
	// graphing price history in Grafana.
	RateValue = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rate_value",
		Help: "Current exchange rate value per currency.",
	}, []string{"currency"})

	// HTTPRequestDuration measures end-to-end request latency.
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
)

// Handler returns the Prometheus metrics HTTP handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Middleware wraps an http.Handler and records latency, method, path and
// response status for every request via HTTPRequestDuration.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		HTTPRequestDuration.
			WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rw.statusCode)).
			Observe(time.Since(start).Seconds())
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code written
// by the downstream handler.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
