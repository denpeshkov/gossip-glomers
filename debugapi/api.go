// Package debugapi provides health check, profiling, and metrics endpoints.
package debugapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RegisterPprof registers pprof debug endpoints on the mux.
func RegisterPprof(mux *http.ServeMux) {
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
}

// RegisterHealthz registers health check endpoints on the mux.
func RegisterHealthz(mux *http.ServeMux, shutdownCh <-chan struct{}) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-shutdownCh:
			http.Error(w, "Shutting down", http.StatusServiceUnavailable)
		default:
			_, _ = fmt.Fprintln(w, http.StatusText(http.StatusOK))
		}
	})
}

// RegisterMetrics registers Prometheus metrics endpoints on the mux.
func RegisterMetrics(mux *http.ServeMux, logger *slog.Logger, reg *prometheus.Registry) {
	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		ErrorLog:            slog.NewLogLogger(logger.Handler(), slog.LevelError),
		Registry:            reg,
		OfferedCompressions: []promhttp.Compression{promhttp.Zstd},
	})
	mux.Handle("GET /metrics", h)
}
