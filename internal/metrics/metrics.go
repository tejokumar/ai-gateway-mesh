package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry *prometheus.Registry

	RequestsTotal            *prometheus.CounterVec
	RequestDuration          *prometheus.HistogramVec
	BackendErrorsTotal       *prometheus.CounterVec
	FallbackTotal            *prometheus.CounterVec
	EstimatedPromptTokens    prometheus.Histogram
	EstimatedCompletionToken prometheus.Histogram
	InflightRequests         prometheus.Gauge
}

func New(registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		registry: registry,
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_gateway_requests_total",
			Help: "Total gateway HTTP requests.",
		}, []string{"endpoint", "backend", "status"}),
		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ai_gateway_request_duration_seconds",
			Help:    "Gateway request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"endpoint", "backend"}),
		BackendErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_gateway_backend_errors_total",
			Help: "Total backend errors observed by the gateway.",
		}, []string{"backend", "status"}),
		FallbackTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ai_gateway_fallback_total",
			Help: "Total backend fallbacks performed by the gateway.",
		}, []string{"from_backend", "to_backend"}),
		EstimatedPromptTokens: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "ai_gateway_estimated_prompt_tokens",
			Help:    "Estimated prompt tokens per chat completion request.",
			Buckets: []float64{1, 16, 64, 128, 256, 512, 1024, 2048, 4096, 8192},
		}),
		EstimatedCompletionToken: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "ai_gateway_estimated_completion_tokens",
			Help:    "Estimated completion tokens per chat completion response.",
			Buckets: []float64{1, 16, 64, 128, 256, 512, 1024, 2048, 4096},
		}),
		InflightRequests: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "ai_gateway_inflight_requests",
			Help: "Current in-flight HTTP requests.",
		}),
	}
	registry.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.BackendErrorsTotal,
		m.FallbackTotal,
		m.EstimatedPromptTokens,
		m.EstimatedCompletionToken,
		m.InflightRequests,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Observe(endpoint, backend string, status int, started time.Time) {
	statusLabel := strconv.Itoa(status)
	m.RequestsTotal.WithLabelValues(endpoint, backend, statusLabel).Inc()
	m.RequestDuration.WithLabelValues(endpoint, backend).Observe(time.Since(started).Seconds())
}
