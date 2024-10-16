package calculator

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"
)

const (
	maxMemory            = 500 * 1024 * 1024 // 500 MB
	largeFactorialNumber = 3e5
	waitTime             = 5 * time.Second
)

type service struct {
	httpClient *http.Client
}

func NewService(httpClient *http.Client) Service {
	return &service{
		httpClient: httpClient,
	}
}

func (s *service) Add(ctx context.Context, a, b int64) (int64, error) {
	largeSlice := make([]byte, maxMemory)

	// This loop simulates holding onto memory without freeing it
	for i := range largeSlice {
		select {
		case <-ctx.Done():

			return 0, ctx.Err()
		default:
			// Simulate work by filling the slice with data
			largeSlice[i] = byte(i)
		}
	}
	if err := errorFunc(); err != nil {
		return 0, err
	}

	return a + b, nil
}

func (s *service) Subtract(ctx context.Context, a, b int64) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
		time.Sleep(waitTime)
	}
	if err := errorFunc(); err != nil {
		return 0, err
	}

	return a - b, nil
}

func (s *service) Multiply(ctx context.Context, a, b int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, waitTime)
	defer cancel()

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
		if err := s.makePizzaRequest(ctx); err != nil {
			return 0, err
		}

		if err := errorFunc(); err != nil {
			return 0, err
		}

		return a * b, nil
	}
}

func (s *service) Divide(_ context.Context, a, b int64) (int64, error) {
	if err := errorFunc(); err != nil {
		return 0, err
	}

	return int64(float64(a) / float64(b)), nil
}

func (s *service) makePizzaRequest(ctx context.Context) error {
	const (
		thirdPartyURL       = "https://quickpizza.grafana.com/api/pizza"
		maxCaloriesPerSlice = 1000
		maxBool             = 2
		maxNumberOfToppings = 10
		minNumberOfToppings = 2
	)

	payload := map[string]interface{}{
		"maxCaloriesPerSlice": generateRandomNumber(maxCaloriesPerSlice),
		"mustBeVegetarian":    generateRandomNumber(maxBool) == 1,
		"excludedIngredients": []string{},
		"excludedTools":       []string{},
		"maxNumberOfToppings": generateRandomNumber(maxNumberOfToppings),
		"minNumberOfToppings": generateRandomNumber(minNumberOfToppings),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, thirdPartyURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("X-User-Id", "298337")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return errors.New("response body is empty")
	}

	return nil
}

func generateRandomNumber(maxNumber int64) int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(maxNumber))
	if err != nil {
		return 0
	}

	return n.Int64()
}

func errorFunc() error {
	if generateRandomNumber(10) > 7 { //nolint:gomnd,mnd // Using 7 as an arbitrary threshold for error generation
		return errors.New("random error")
	}

	return nil
}
