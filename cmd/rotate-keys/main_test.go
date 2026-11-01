package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("Error/MasterKey", func(t *testing.T) {
		t.Parallel()

		cfg := config.JobRotateKeys{App: config.Main{MasterKey: "not hex"}}

		require.ErrorContains(t, run(t.Context(), cfg, time.Now()), "load master key")
	})
}
