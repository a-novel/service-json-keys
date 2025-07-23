package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/a-novel/service-json-keys/models/api"
)

const (
	defaultPingInterval = time.Second
	defaultPingTimeout  = 16 * time.Second
)

// NewAPIClient creates a new client to interact with a JSON keys server.
func NewAPIClient(ctx context.Context, url string) (*apimodels.Client, error) {
	client, err := apimodels.NewClient(url)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	start := time.Now()
	_, err = client.Healthcheck(ctx)

	for time.Since(start) < defaultPingTimeout && err != nil {
		time.Sleep(defaultPingInterval)

		_, err = client.Healthcheck(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("ping server: %w", err)
	}

	return client, nil
}
