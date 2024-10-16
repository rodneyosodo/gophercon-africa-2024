package middleware

import (
	"context"

	"github.com/rodneyosodo/gophercon/calculator"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ calculator.Service = (*tracing)(nil)

type tracing struct {
	tracer trace.Tracer
	svc    calculator.Service
}

func Tracing(tracer trace.Tracer, svc calculator.Service) calculator.Service {
	return &tracing{tracer, svc}
}

func (t *tracing) Add(ctx context.Context, a, b int64) (result int64, err error) {
	defer t.trace(ctx, "calculator.Add", a, b, result, err)

	return t.svc.Add(ctx, a, b)
}

func (t *tracing) Subtract(ctx context.Context, a, b int64) (result int64, err error) {
	defer t.trace(ctx, "calculator.Subtract", a, b, result, err)

	return t.svc.Subtract(ctx, a, b)
}

func (t *tracing) Multiply(ctx context.Context, a, b int64) (result int64, err error) {
	defer t.trace(ctx, "calculator.Multiply", a, b, result, err)

	return t.svc.Multiply(ctx, a, b)
}

func (t *tracing) Divide(ctx context.Context, a, b int64) (result int64, err error) {
	defer t.trace(ctx, "calculator.Divide", a, b, result, err)

	return t.svc.Divide(ctx, a, b)
}

func (t *tracing) trace(ctx context.Context, name string, a, b, result int64, err error) {
	attributes := []attribute.KeyValue{
		attribute.Int64("a", a),
		attribute.Int64("b", b),
		attribute.Int64("result", result),
	}
	if err != nil {
		attributes = append(attributes, attribute.String("error", err.Error()))
	}
	_, span := t.tracer.Start(ctx, name, trace.WithAttributes(attributes...))

	span.End()
}
