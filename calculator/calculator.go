package calculator

import "context"

// Service is the interface for the calculator service.
// It defines the operations that can be performed on the calculator.
// These include addition, subtraction, multiplication, and division.
type Service interface {
	Add(ctx context.Context, a, b int64) (int64, error)
	Subtract(ctx context.Context, a, b int64) (int64, error)
	Multiply(ctx context.Context, a, b int64) (int64, error)
	Divide(ctx context.Context, a, b int64) (int64, error)
}
