package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
)

const (
	defaultPingInterval = 500 * time.Millisecond
	defaultPingTimeout  = 16 * time.Second
)

type APIClient = codegen.Client

// NewAPIClient creates a new client to interact with a JSON keys server.
func NewAPIClient(ctx context.Context, url string) (*APIClient, error) {
	client, err := codegen.NewClient(url)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	start := time.Now()
	_, err = client.Ping(ctx)

	for time.Since(start) < defaultPingTimeout && err != nil {
		time.Sleep(defaultPingInterval)

		_, err = client.Ping(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("ping server: %w", err)
	}

	return client, nil
}
