package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RegisterPprof(mux *http.ServeMux) {
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
}

func RegisterHealthz(mux *http.ServeMux, sig <-chan struct{}) {
	h := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-sig:
			http.Error(w, "Shutting down", http.StatusServiceUnavailable)
		default:
			_, _ = fmt.Fprintln(w, http.StatusText(http.StatusOK))
		}
	}
	mux.HandleFunc("GET /healthz", h)
}

func RegisterMetrics(mux *http.ServeMux, logger *slog.Logger, reg *prometheus.Registry) {
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{ //nolint:exhaustruct
		ErrorLog:            slog.NewLogLogger(logger.Handler(), slog.LevelError),
		Registry:            reg,
		OfferedCompressions: []promhttp.Compression{promhttp.Zstd},
	})
	mux.Handle("GET /metrics", h)
}

type Metric struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

func NewMetric(factory promauto.Factory) Metric {
	requestsTotal := factory.NewCounterVec(
		prometheus.CounterOpts{ //nolint:exhaustruct
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		}, []string{"method", "route", "code"},
	)
	requestDuration := factory.NewHistogramVec(
		prometheus.HistogramOpts{ //nolint:exhaustruct
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of latencies for HTTP requests",
			Buckets: []float64{.1, .2, .4, 1, 3, 8, 20, 60, 120},
		},
		[]string{"method", "route", "code"},
	)
	return Metric{
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
	}
}
