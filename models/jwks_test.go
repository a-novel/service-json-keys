package models_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/models"
)

func TestJWKConfig(t *testing.T) {
	t.Parallel()

	for _, usage := range models.KnownKeyUsages {
		require.NotEmpty(t, usage.String())

		cfg, ok := models.DefaultJWKSConfig[usage]
		require.True(t, ok)
		require.NotNil(t, cfg)
	}
}
