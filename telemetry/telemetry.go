// Package telemetry provides logging, metrics, and tracing setup.
package telemetry

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Config holds telemetry subsystem configurations.
type Config struct {
	Tracer TracerConfig
	Logger LoggerConfig
	Meter  MeterConfig
}

// Telemetry bundles logging, metrics, and tracing components.
type Telemetry struct {
	Tracer *Tracer
	Logger *slog.Logger
	Meter  promauto.Factory
}

// NewNoopTelemetry creates a Telemetry with no-op implementations.
func NewNoopTelemetry() Telemetry {
	return Telemetry{
		Tracer: NewNoopTracer(),
		Logger: NewNoopLogger(),
		Meter:  NewNoopMeter(),
	}
}
