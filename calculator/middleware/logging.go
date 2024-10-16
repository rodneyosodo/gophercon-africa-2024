package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/rodneyosodo/gophercon/calculator"
)

var _ calculator.Service = (*logging)(nil)

type logging struct {
	logger *slog.Logger
	svc    calculator.Service
}

func Logging(logger *slog.Logger, svc calculator.Service) calculator.Service {
	return &logging{
		logger: logger,
		svc:    svc,
	}
}

func (l *logging) Add(ctx context.Context, a, b int64) (result int64, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Int64("a", a),
			slog.Int64("b", b),
		}

		switch err {
		case nil:
			l.logger.InfoContext(ctx, "Add completed successfully", args...)
		default:
			args = append(args, slog.String("error", err.Error()))
			l.logger.WarnContext(ctx, "Add failed", args...)
		}
	}(time.Now())

	return l.svc.Add(ctx, a, b)
}

func (l *logging) Subtract(ctx context.Context, a, b int64) (result int64, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Int64("a", a),
			slog.Int64("b", b),
		}

		switch err {
		case nil:
			l.logger.InfoContext(ctx, "Subtract completed successfully", args...)
		default:
			args = append(args, slog.String("error", err.Error()))
			l.logger.WarnContext(ctx, "Subtract failed", args...)
		}
	}(time.Now())

	return l.svc.Subtract(ctx, a, b)
}

func (l *logging) Multiply(ctx context.Context, a, b int64) (result int64, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Int64("a", a),
			slog.Int64("b", b),
		}

		switch err {
		case nil:
			l.logger.InfoContext(ctx, "Multiply completed successfully", args...)
		default:
			args = append(args, slog.String("error", err.Error()))
			l.logger.WarnContext(ctx, "Multiply failed", args...)
		}
	}(time.Now())

	return l.svc.Multiply(ctx, a, b)
}

func (l *logging) Divide(ctx context.Context, a, b int64) (result int64, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Int64("a", a),
			slog.Int64("b", b),
		}

		switch err {
		case nil:
			l.logger.InfoContext(ctx, "Divide completed successfully", args...)
		default:
			args = append(args, slog.String("error", err.Error()))
			l.logger.WarnContext(ctx, "Divide failed", args...)
		}
	}(time.Now())

	return l.svc.Divide(ctx, a, b)
}
