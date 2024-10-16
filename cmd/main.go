package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

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

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var g *errgroup.Group
	g, _ = errgroup.WithContext(ctx)

	logLevel := flag.String("log_level", "info", "log level")
	addr := flag.String("addr", ":11211", "gophercon calculator server address")
	prometheusEndpoint := flag.String("prometheus_endpoint", ":9464", "the Prometheus exporter endpoint")
	readTimeout := flag.Duration("read_timeout", defHTTPTimeout, "the read timeout for the server")
	writeTimeout := flag.Duration("write_timeout", defHTTPTimeout, "the write timeout for the server")

	flag.Parse()

	var level slog.Level
	if err := level.UnmarshalText([]byte(*logLevel)); err != nil {
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
			Addr:         *prometheusEndpoint,
			Handler:      promhttp.Handler(),
			ReadTimeout:  *readTimeout,
			WriteTimeout: *writeTimeout,
		}

		return server.ListenAndServe()
	})
	logger.Info("Prometheus exporter started", slog.String("endpoint", *prometheusEndpoint))

	so := opentelemetry.ServerOption(
		opentelemetry.Options{MetricsOptions: opentelemetry.MetricsOptions{MeterProvider: provider}},
	)

	listener, err := net.Listen("tcp", *addr)
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

	logger.Info("Calculator server started", slog.String("address", *addr))

	if err := g.Wait(); err != nil {
		logger.Error("Failed to serve", slog.String("error", err.Error()))
		cancel()
		os.Exit(1)
	}
}
