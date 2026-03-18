package telemetry

import (
	"errors"
	"flag"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MeterConfig holds Prometheus metrics settings.
type MeterConfig struct {
	Namespace string `env:"PROMETHEUS_NAMESPACE"`
}

func (c *MeterConfig) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Namespace, "prometheus_namespace", "", "Prometheus metrics namespace (env: PROMETHEUS_NAMESPACE)")
}

// NewMeter creates a Prometheus metrics factory.
func NewMeter(cfg MeterConfig) (promauto.Factory, error) {
	reg := prometheus.NewRegistry()
	if err := reg.Register(collectors.NewGoCollector()); err != nil {
		return promauto.Factory{}, errors.New("register go collector")
	}
	if err := reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
		Namespace: cfg.Namespace,
	})); err != nil {
		return promauto.Factory{}, errors.New("register process collector")
	}
	return promauto.With(reg), nil
}

// NewNoopMeter creates a metrics factory that discards all metrics.
func NewNoopMeter() promauto.Factory {
	return promauto.Factory{}
}
