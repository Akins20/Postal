// Package metrics provides Postal's Prometheus instrumentation: a private
// registry, base HTTP counters/histograms, request-recording middleware, and an
// exposition handler. The registry is owned by a Metrics value (no global
// state) and injected where needed.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Postal's collectors and their registry.
type Metrics struct {
	reg          *prometheus.Registry
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
}

// New builds a Metrics with a fresh registry, the base HTTP collectors, and the
// standard Go runtime + process collectors registered.
func New() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		reg: reg,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "postal_http_requests_total",
			Help: "Total HTTP requests by method, route, and status.",
		}, []string{"method", "route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "postal_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds by method and route.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
	}
	reg.MustRegister(m.httpRequests, m.httpDuration)
	return m
}

// Handler serves the metrics exposition endpoint for this registry.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// Middleware records request count and latency. It labels by the chi route
// pattern (not the raw path) to keep cardinality bounded.
func (m *Metrics) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			route := routePattern(r)
			m.httpRequests.WithLabelValues(r.Method, route, strconv.Itoa(ww.Status())).Inc()
			m.httpDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}

// routePattern returns the matched chi route template (e.g. "/api/v1/posts/{id}")
// or "unmatched" when no route matched, bounding label cardinality.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return "unmatched"
}
