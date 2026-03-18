package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"

	"github.com/denpeshkov/go-template/telemetry"
	"github.com/denpeshkov/go-template/web"
)

type config struct {
	InfraAddr string `env:"INFRA_HTTP_ADDRESS" envDefault:":6060"`
	APIAddr   string `env:"API_HTTP_ADDRESS"   envDefault:":8080"`
	HTTP      struct {
		ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"1s"`
		ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT"        envDefault:"3s"`
		WriteTimeout      time.Duration `env:"HTTP_WRTIE_TIMEOUT"       envDefault:"3s"`
		IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT"        envDefault:"1m"`
	}
	DB struct {
		URL string `env:"POSTGRESQL_URL,required"`
	}
	ReadinessCheckTimeout time.Duration `env:"READINESS_CHECK_TIMEOUT" envDefault:"10s"`
	Logger                telemetry.LoggerConfig
	Meter                 telemetry.MeterConfig
	Tracer                telemetry.TracerConfig
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	sigCtx, sigStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer sigStop()

	var config config
	if err := env.Parse(&config); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	logger := telemetry.NewLogger(config.Logger)
	meter, err := telemetry.NewMeter(config.Meter)
	if err != nil {
		return fmt.Errorf("create meter: %w", err)
	}
	tracer, tracingShutdown, err := telemetry.NewTracer(sigCtx, config.Tracer)
	if err != nil {
		return fmt.Errorf("create otel tracer: %w", err)
	}
	telem := telemetry.Telemetry{
		Tracer: tracer,
		Logger: logger,
		Meter:  meter,
	}
	_ = telem // FIXME:

	httpBaseCtx, httpBaseCtxCancel := context.WithCancel(context.Background())
	defer httpBaseCtxCancel()

	promReg := prometheus.NewRegistry()

	infraMux := http.NewServeMux() // TODO: Move mux to server?
	web.RegisterHealthz(infraMux, sigCtx.Done())
	web.RegisterPprof(infraMux)
	web.RegisterMetrics(infraMux, logger, promReg)
	infraServer := http.Server{ //nolint:exhaustruct
		Addr:              config.InfraAddr,
		Handler:           infraMux,
		ReadHeaderTimeout: config.HTTP.ReadHeaderTimeout,
		ReadTimeout:       config.HTTP.ReadTimeout,
		IdleTimeout:       config.HTTP.IdleTimeout,
		BaseContext:       func(net.Listener) context.Context { return httpBaseCtx },
	}

	eg, egCtx := errgroup.WithContext(sigCtx)

	// Start services.
	eg.Go(func() error {
		logger.Info("starting infra server", "addr", infraServer.Addr)
		if err := infraServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve infra server: %w", err)
		}
		return nil
	})

	// Wait for a signal and shutdown services.
	eg.Go(func() error {
		<-egCtx.Done()

		sigStop() // Allow second SIGINT to forcefully terminate.

		// Give time for readiness check to propagate.
		time.Sleep(config.ReadinessCheckTimeout)
		logger.Info("readiness check propagated, waiting for ongoing http requests to finish")

		shutCtx, shutCancel := context.WithTimeout(context.WithoutCancel(egCtx), 15*time.Second)
		defer shutCancel()

		// Signal http handlers to finish.
		httpBaseCtxCancel()

		if err := infraServer.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("shutdown infra server: %w", err)
		}
		logger.Info("infra server shutdown")

		if err := tracingShutdown(shutCtx); err != nil {
			return fmt.Errorf("shutdown tracing: %w", err)
		}
		logger.Info("tracing shutdown")

		return nil
	})

	return eg.Wait()
}
