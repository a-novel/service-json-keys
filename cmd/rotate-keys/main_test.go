package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/sentryf/sentryftest"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

//nolint:paralleltest // Records through the process-wide Sentry reporter.
func TestRun(t *testing.T) {
	t.Run("Error/MasterKey", func(t *testing.T) {
		transport := sentryftest.Record(t)

		cfg := config.JobRotateKeys{App: config.Main{MasterKey: "not hex"}}

		require.ErrorContains(t, run(t.Context(), cfg, time.Now()), "load master key")

		events := transport.Events()
		require.Len(t, events, 1)
		require.Equal(t, "rotate-keys", events[0].Exception[0].Value)
	})
}
