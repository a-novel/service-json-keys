package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/config"
)

func TestJWKConfig(t *testing.T) {
	t.Parallel()

	for _, usage := range models.KnownKeyUsages {
		require.NotEmpty(t, usage.String())

		cfg, ok := config.JWKSPresetDefault[usage]
		require.True(t, ok)
		require.NotNil(t, cfg)
	}
}
