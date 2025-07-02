package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/pkg"
)

// STORY: The user can call health apis, and they return a 200 status code.

func TestHealthAPI(t *testing.T) {
	client, err := pkg.NewAPIClient(context.Background(), fmt.Sprintf("http://127.0.0.1:%v/v1", config.API.Port))
	require.NoError(t, err)

	_, err = client.Ping(t.Context())
	require.NoError(t, err)

	_, err = client.Healthcheck(t.Context())
	require.NoError(t, err)
}
