package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rodneyosodo/gophercon/calculator"
	"github.com/rodneyosodo/gophercon/calculator/api"
	"github.com/rodneyosodo/gophercon/calculator/middleware"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats/opentelemetry"
)

const defHTTPTimeout = 10 * time.Second

type config struct {
	LogLevel           string        `env:"GOPHERCON_LOG_LEVEL"           envDefault:"info"`
	Addr               string        `env:"GOPHERCON_ADDR"                envDefault:":11211"`
	PrometheusEndpoint string        `env:"GOPHERCON_PROMETHEUS_ENDPOINT" envDefault:":9464"`
	ReadTimeout        time.Duration `env:"GOPHERCON_READ_TIMEOUT"        envDefault:"10s"`
	WriteTimeout       time.Duration `env:"GOPHERCON_WRITE_TIMEOUT"       envDefault:"10s"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var g *errgroup.Group
	g, _ = errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load configuration : %s", err.Error())
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		log.Fatalf("failed to parse log level: %s", err.Error())
	}
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	exporter, err := prometheus.New()
	if err != nil {
		logger.Error("Failed to start prometheus exporter", slog.String("error", err.Error()))
		cancel()
		os.Exit(1)
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))

	g.Go(func() error {
		server := &http.Server{
			Addr:         cfg.PrometheusEndpoint,
			Handler:      promhttp.Handler(),
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		}

		return server.ListenAndServe()
	})
	logger.Info("Prometheus exporter started", slog.String("endpoint", cfg.PrometheusEndpoint))

	so := opentelemetry.ServerOption(
		opentelemetry.Options{MetricsOptions: opentelemetry.MetricsOptions{MeterProvider: provider}},
	)

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		logger.Error("Failed to listen", slog.String("error", err.Error()))
		cancel()
		os.Exit(1)
	}

	server := grpc.NewServer(so)
	reflection.Register(server)

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	retryClient.Logger = logger
	httpClient := retryClient.StandardClient()

	service := calculator.NewService(httpClient)
	service = middleware.Logging(logger, service)
	calculator.RegisterCalculatorServer(server, api.NewGrpcServer(service))

	g.Go(func() error {
		return server.Serve(listener)
	})

	logger.Info("Calculator server started", slog.String("address", cfg.Addr))

	if err := g.Wait(); err != nil {
		logger.Error("Failed to serve", slog.String("error", err.Error()))
		cancel()
		os.Exit(1)
	}
}
