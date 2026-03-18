package telemetry

import (
	"context"
	"flag"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"gitlab.semrush.net/elysium/ai-website-optimization-co-pilot/cortex/internal/buildinfo"
)

// TracerConfig holds OpenTelemetry tracing settings.
type TracerConfig struct {
	Enabled          bool          `env:"OTEL_ENABLED"                                envDefault:"true"`
	URL              string        `env:"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"          envDefault:"opentelemetry-collector.observability.svc:4317"`
	Timeout          time.Duration `env:"OTEL_EXPORTER_OTLP_TRACES_TIMEOUT"           envDefault:"10s"`
	SamplingFraction float64       `env:"OTEL_EXPORTER_OTLP_TRACES_SAMPLING_FRACTION" envDefault:"1"`
}

func (c *TracerConfig) RegisterFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.Enabled, "otel_enabled", true, "Enable OpenTelemetry tracing (env: OTEL_ENABLED)")
	fs.StringVar(&c.URL, "otel_endpoint", "opentelemetry-collector.observability.svc:4317", "OTLP traces endpoint (env: OTEL_EXPORTER_OTLP_TRACES_ENDPOINT)")
	fs.DurationVar(&c.Timeout, "otel_timeout", 10*time.Second, "OTLP export timeout (env: OTEL_EXPORTER_OTLP_TRACES_TIMEOUT)")
	fs.Float64Var(&c.SamplingFraction, "otel_sampling", 1.0, "Trace sampling fraction 0.0-1.0 (env: OTEL_EXPORTER_OTLP_TRACES_SAMPLING_FRACTION)")
}

// Tracer wraps an OpenTelemetry tracer provider.
type Tracer struct {
	trace.Tracer
	tp trace.TracerProvider
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	if tp, ok := t.tp.(interface {
		Shutdown(ctx context.Context) error
	}); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}

// NewTracer creates a Tracer with the given configuration.
func NewTracer(ctx context.Context, cfg TracerConfig) (*Tracer, error) {
	rs, err := newResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTel resource: %w", err)
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpointURL(cfg.URL),
		otlptracegrpc.WithTimeout(cfg.Timeout),
		otlptracegrpc.WithCompressor("gzip"),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(rs),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplingFraction))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	tracer := tp.Tracer(buildinfo.Name, trace.WithSchemaURL(semconv.SchemaURL))
	return &Tracer{Tracer: tracer, tp: tp}, nil
}

// NewNoopTracer creates a tracer that discards all spans.
func NewNoopTracer() *Tracer {
	return &Tracer{Tracer: noop.Tracer{}, tp: noop.NewTracerProvider()}
}

func newResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithHost(),
		resource.WithContainerID(),
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(buildinfo.Name),
			semconv.ServiceVersion(buildinfo.Version),
		),
	)
}
