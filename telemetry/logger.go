package telemetry

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// LoggerConfig holds structured logging settings.
type LoggerConfig struct {
	AddSource bool      `env:"LOG_ADD_SOURCE" envDefault:"false"`
	Level     string    `env:"LOG_LEVEL"      envDefault:"error"`
	Writer    io.Writer // Output destination, os.Stderr by default.
}

func (c *LoggerConfig) RegisterFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.AddSource, "log_add_source", false, "Add source code position to log output (env: LOG_ADD_SOURCE)")
	fs.StringVar(&c.Level, "log_level", "error", "Log level: debug, info, warn, error (env: LOG_LEVEL)")
}

// NewLogger creates a slog.Logger with the given configuration.
func NewLogger(cfg LoggerConfig) *slog.Logger {
	levelLowerCase := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey {
			return slog.Attr{
				Key:   a.Key,
				Value: slog.StringValue(strings.ToLower(a.Value.String())),
			}
		}
		return a
	}

	if cfg.Writer == nil {
		cfg.Writer = os.Stderr
	}

	opts := &slog.HandlerOptions{
		AddSource:   cfg.AddSource,
		Level:       level(cfg.Level),
		ReplaceAttr: levelLowerCase,
	}
	return slog.New(otelHandler{slog.NewJSONHandler(cfg.Writer, opts)})
}

// NewNoopLogger creates a logger that discards all output.
func NewNoopLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

type otelHandler struct {
	slog.Handler
}

func (h otelHandler) Handle(ctx context.Context, record slog.Record) error {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		record.AddAttrs(slog.String("trace_id", spanCtx.TraceID().String()))
	}
	if spanCtx.HasSpanID() {
		record.AddAttrs(slog.String("span_id", spanCtx.SpanID().String()))
	}
	return h.Handler.Handle(ctx, record)
}

func (h otelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return otelHandler{h.Handler.WithAttrs(attrs)}
}

func (h otelHandler) WithGroup(name string) slog.Handler {
	return otelHandler{h.Handler.WithGroup(name)}
}

func level(lvl string) slog.Level {
	switch strings.ToLower(lvl) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
