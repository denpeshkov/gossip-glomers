package telemetrytest

import (
	"context"
	"log/slog"
	"testing"
)

// NewTestLogger returns a [slog.Logger] that writes directly to [t.Output]
// and fails the test if an error log is emitted.
func NewTestLogger(tb testing.TB) *slog.Logger {
	tb.Helper()

	h := slog.NewTextHandler(tb.Output(), &slog.HandlerOptions{
		Level: slog.LevelDebug,
		// Remove timestamp.
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	return slog.New(&failOnErrorHandler{Handler: h, t: tb})
}

type failOnErrorHandler struct {
	slog.Handler
	t testing.TB
}

func (h *failOnErrorHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.Handler.Handle(ctx, r)
	if r.Level >= slog.LevelError {
		h.t.Fail()
	}
	return err
}
