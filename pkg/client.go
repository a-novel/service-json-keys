package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
)

func NewAPIClient(url string) (*codegen.Client, error) {
	client, err := codegen.NewClient(url)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	start := time.Now()
	_, err = client.Ping(context.Background())

	for time.Since(start) < 16*time.Second && err != nil {
		_, err = client.Ping(context.Background())
	}

	if err != nil {
		return nil, fmt.Errorf("ping server: %w", err)
	}

	return client, nil
}
