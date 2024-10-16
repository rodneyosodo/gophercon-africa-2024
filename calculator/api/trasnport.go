package api

import (
	"context"

	"github.com/rodneyosodo/gophercon/calculator"
)

var _ calculator.CalculatorServer = (*grpcServer)(nil)

type grpcServer struct {
	calculator.UnimplementedCalculatorServer
	service calculator.Service
}

func NewGrpcServer(service calculator.Service) calculator.CalculatorServer {
	return &grpcServer{service: service}
}

func (s *grpcServer) Add(ctx context.Context, req *calculator.Request) (*calculator.Response, error) {
	result, err := s.service.Add(ctx, req.GetA(), req.GetB())
	if err != nil {
		return nil, err
	}

	return &calculator.Response{Result: result}, nil
}

func (s *grpcServer) Subtract(ctx context.Context, req *calculator.Request) (*calculator.Response, error) {
	result, err := s.service.Subtract(ctx, req.GetA(), req.GetB())
	if err != nil {
		return nil, err
	}

	return &calculator.Response{Result: result}, nil
}

func (s *grpcServer) Multiply(ctx context.Context, req *calculator.Request) (*calculator.Response, error) {
	result, err := s.service.Multiply(ctx, req.GetA(), req.GetB())
	if err != nil {
		return nil, err
	}

	return &calculator.Response{Result: result}, nil
}

func (s *grpcServer) Divide(ctx context.Context, req *calculator.Request) (*calculator.Response, error) {
	result, err := s.service.Divide(ctx, req.GetA(), req.GetB())
	if err != nil {
		return nil, err
	}

	return &calculator.Response{Result: result}, nil
}
