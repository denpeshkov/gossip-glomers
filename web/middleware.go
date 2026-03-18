package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/denpeshkov/go-template/telemetry"
)

func RecoverPanic(tlm telemetry.Telemetry) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.Header().Set("Connection", "close")
					tlm.Logger.ErrorContext(r.Context(), "panic during request processing", "error", err)
				}
			}()
			h.ServeHTTP(w, r)
		})
	}
}

func InstrumentHandler(metric Metric) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &statusResponseWriter{w, http.StatusOK}
			h.ServeHTTP(rw, r)

			reqDur := time.Since(start)
			code := strconv.Itoa(rw.statusCode)

			metric.RequestDuration.WithLabelValues(r.Method, r.Pattern, code).Observe(reqDur.Seconds())
			metric.RequestsTotal.WithLabelValues(r.Method, r.Pattern, code).Add(1)
		})
	}
}

// statusResponseWriter records status of the HTTP response.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Unwrap is used by [http.ResponseController].
func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
