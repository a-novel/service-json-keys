package handlers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
)

// newPostgresContext returns the test context, holding the PostgreSQL connection of the integration
// tests.
func newPostgresContext(t *testing.T) context.Context {
	t.Helper()

	ctx, err := postgres.NewContext(t.Context(), configtest.PostgresPreset)
	require.NoError(t, err)

	return ctx
}
