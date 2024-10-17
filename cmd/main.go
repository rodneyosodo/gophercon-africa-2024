package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/pyroscope-go"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rodneyosodo/gophercon/calculator"
	"github.com/rodneyosodo/gophercon/calculator/api"
	"github.com/rodneyosodo/gophercon/calculator/middleware"
	slogloki "github.com/samber/slog-loki/v3"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats/opentelemetry"
)

type config struct {
	LogLevel           string        `env:"GOPHERCON_LOG_LEVEL"           envDefault:"info"`
	Addr               string        `env:"GOPHERCON_ADDR"                envDefault:":6000"`
	PrometheusEndpoint string        `env:"GOPHERCON_PROMETHEUS_ENDPOINT" envDefault:":6001"`
	ReadTimeout        time.Duration `env:"GOPHERCON_READ_TIMEOUT"        envDefault:"10s"`
	WriteTimeout       time.Duration `env:"GOPHERCON_WRITE_TIMEOUT"       envDefault:"10s"`
	OTELURL            url.URL       `env:"GOPHERCON_OTEL_URL"            envDefault:""`
	TraceRatio         float64       `env:"GOPHERCON_TRACE_RATIO"         envDefault:"0.1"`
	LokiURL            string        `env:"GOPHERCON_LOKI_URL"            envDefault:""`
	PyroScopeURL       string        `env:"GOPHERCON_PYROSCOPE_URL"       envDefault:""`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load configuration : %s", err.Error())
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		log.Fatalf("failed to parse log level: %s", err.Error())
	}
	fanout := slogmulti.Fanout(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		}),
	)
	if cfg.LokiURL != "" {
		config, err := loki.NewDefaultConfig(cfg.LokiURL)
		if err != nil {
			log.Fatalf("failed to create loki config: %s", err.Error())
		}
		config.TenantID = "gophercon"
		client, err := loki.New(config)
		if err != nil {
			log.Fatalf("failed to create loki client: %s", err.Error())
		}

		hander := slogloki.Option{Level: level, Client: client}.NewLokiHandler()
		fanout = slogmulti.Fanout(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
			}),
			hander,
		)
	}

	logger := slog.New(fanout).With("service", "gophercon")
	slog.SetDefault(logger)

	if cfg.PyroScopeURL != "" {
		if _, err := pyroscope.Start(pyroscope.Config{
			ApplicationName: "gophercon",
			ServerAddress:   cfg.PyroScopeURL,
			Logger:          nil,
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
			},
		}); err != nil {
			log.Fatalf("failed to start pyroscope: %s", err.Error())
		}
	}

	var tp trace.TracerProvider
	switch {
	case cfg.OTELURL == (url.URL{}):
		tp = noop.NewTracerProvider()
	default:
		sdktp, err := initTracer(ctx, cfg.OTELURL, cfg.TraceRatio)
		if err != nil {
			log.Fatalf("failed to initialize opentelemetry: %s", err.Error())
		}
		defer func() {
			if err := sdktp.Shutdown(ctx); err != nil {
				log.Fatalf("error shutting down tracer provider: %s", err.Error())
			}
		}()
		tp = sdktp
	}
	tracer := tp.Tracer("gophercon")

	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("Failed to start prometheus exporter: %s", err.Error())
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
		log.Fatalf("Failed to listen: %s", err.Error())
	}

	server := grpc.NewServer(so, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	reflection.Register(server)

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	retryClient.Logger = logger
	httpClient := retryClient.StandardClient()

	service := calculator.NewService(httpClient)
	service = middleware.Logging(logger, service)
	service = middleware.Tracing(tracer, service)
	calculator.RegisterCalculatorServer(server, api.NewGrpcServer(service))

	g.Go(func() error {
		return server.Serve(listener)
	})

	logger.Info("Calculator server started", slog.String("address", cfg.Addr))

	if err := g.Wait(); err != nil {
		log.Fatalf("Failed to serve: %s", err.Error())
	}
}

func initTracer(ctx context.Context, otelURL url.URL, fraction float64) (*sdktrace.TracerProvider, error) {
	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(otelURL.Host), otlptracehttp.WithURLPath(otelURL.Path),
	}

	var client otlptrace.Client
	switch otelURL.Scheme {
	case "http":
		options = append(options, otlptracehttp.WithInsecure())
		client = otlptracehttp.NewClient(options...)
	case "https":
		client = otlptracehttp.NewClient(options...)
	}

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String("gophercon"),
	}

	hostAttr, err := resource.New(ctx, resource.WithHost(), resource.WithOSDescription(), resource.WithContainer())
	if err != nil {
		return nil, err
	}
	attributes = append(attributes, hostAttr.Attributes()...)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(fraction)),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attributes...,
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
