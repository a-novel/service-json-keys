package cmdpkg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/models/api"
)

func testAppPing(_ context.Context, t *testing.T, client *apimodels.Client) {
	t.Helper()

	_, err := client.Ping(t.Context())
	require.NoError(t, err)

	_, err = client.Healthcheck(t.Context())
	require.NoError(t, err)
}
