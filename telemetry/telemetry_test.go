package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/nalgeon/be"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

func TestLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"Debug", slog.LevelDebug},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"invalid", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := level(tt.input)
			be.Equal(t, got, tt.want)
		})
	}
}

func TestNewNoopLogger(t *testing.T) {
	t.Parallel()

	logger := NewNoopLogger()
	be.True(t, logger != nil)

	// Logging should not panic.
	logger.Info("test message")
	logger.Error("error message", "key", "value")
}

func TestOTelHandler_Handle_WithTraceContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	handler := otelHandler{slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})}
	logger := slog.New(handler)

	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	be.Err(t, err, nil)
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	be.Err(t, err, nil)

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

	logger.InfoContext(ctx, "test message")

	var logEntry map[string]any
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	be.Err(t, err, nil)
	be.Equal(t, logEntry["trace_id"], "0102030405060708090a0b0c0d0e0f10")
	be.Equal(t, logEntry["span_id"], "0102030405060708")
	be.Equal(t, logEntry["msg"], "test message")
}

func TestOTelHandler_Handle_WithoutTraceContext(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	handler := otelHandler{slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})}
	logger := slog.New(handler)

	logger.InfoContext(context.Background(), "test message")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	be.Err(t, err, nil)

	_, hasTraceID := logEntry["trace_id"]
	_, hasSpanID := logEntry["span_id"]
	be.True(t, !hasTraceID)
	be.True(t, !hasSpanID)
	be.Equal(t, logEntry["msg"], "test message")
}

func TestOTelHandler_WithAttrs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := otelHandler{baseHandler}

	newHandler := handler.WithAttrs([]slog.Attr{slog.String("service", "test")})

	_, ok := newHandler.(otelHandler)
	be.True(t, ok)

	logger := slog.New(newHandler)
	logger.Info("test message")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	be.Err(t, err, nil)
	be.Equal(t, logEntry["service"], "test")
}

func TestOTelHandler_WithGroup(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	baseHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler := otelHandler{baseHandler}

	newHandler := handler.WithGroup("request")

	_, ok := newHandler.(otelHandler)
	be.True(t, ok)

	logger := slog.New(newHandler)
	logger.Info("test message", "method", "GET")

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	be.Err(t, err, nil)

	requestGroup, ok := logEntry["request"].(map[string]any)
	be.True(t, ok)
	be.Equal(t, requestGroup["method"], "GET")
}

func TestNewMeter(t *testing.T) {
	t.Parallel()

	t.Run("empty namespace", func(t *testing.T) {
		t.Parallel()

		factory, err := NewMeter(MeterConfig{})
		be.Err(t, err, nil)

		counter := factory.NewCounter(prometheus.CounterOpts{
			Name: "test_requests_total",
			Help: "Test counter",
		})
		counter.Inc()
	})

	t.Run("with namespace", func(t *testing.T) {
		t.Parallel()

		factory, err := NewMeter(MeterConfig{Namespace: "myapp"})
		be.Err(t, err, nil)

		counter := factory.NewCounter(prometheus.CounterOpts{
			Name: "test_requests_total",
			Help: "Test counter",
		})
		counter.Inc()
	})
}

func TestNewNoopMeter(t *testing.T) {
	t.Parallel()

	// NewNoopMeter returns an empty factory that does not panic.
	_ = NewNoopMeter()
}

func TestNewNoopTracer(t *testing.T) {
	t.Parallel()

	tracer := NewNoopTracer()
	be.True(t, tracer != nil)
	be.True(t, tracer.Tracer != nil)
	be.True(t, tracer.tp != nil)

	// NewNoopTracer creates spans without error.
	_, span := tracer.Start(context.Background(), "test-span")
	be.True(t, span != nil)
	span.End()
}

func TestTracer_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("noop tracer", func(t *testing.T) {
		t.Parallel()

		tracer := NewNoopTracer()
		err := tracer.Shutdown(context.Background())
		be.Err(t, err, nil)
	})

	t.Run("noop tracer with cancelled context", func(t *testing.T) {
		t.Parallel()

		tracer := NewNoopTracer()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := tracer.Shutdown(ctx)
		be.Err(t, err, nil)
	})
}

func TestNewNoopTelemetry(t *testing.T) {
	t.Parallel()

	tel := NewNoopTelemetry()

	be.True(t, tel.Tracer != nil)
	be.True(t, tel.Logger != nil)

	// Logger logs without panic.
	tel.Logger.Info("test message")

	// Tracer creates spans without error.
	_, span := tel.Tracer.Start(context.Background(), "test")
	span.End()

	// Shutdown returns nil for noop tracer.
	err := tel.Tracer.Shutdown(context.Background())
	be.Err(t, err, nil)
}
