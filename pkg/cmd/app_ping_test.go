package cmdpkg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
)

func testAppPing(_ context.Context, t *testing.T, client *codegen.Client) {
	t.Helper()

	_, err := client.Ping(t.Context())
	require.NoError(t, err)

	_, err = client.Healthcheck(t.Context())
	require.NoError(t, err)
}
