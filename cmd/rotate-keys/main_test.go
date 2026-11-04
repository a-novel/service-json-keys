package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gootel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/a-novel-kit/golib/otel"

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

//nolint:paralleltest // Replaces the process-wide tracer provider while the job runs.
func TestRunTelemetry(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := gootel.GetTracerProvider()

	gootel.SetTracerProvider(provider)
	t.Cleanup(func() {
		gootel.SetTracerProvider(previous)
		require.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context())))
	})

	cfg := config.JobRotateKeys{App: config.Main{MasterKey: "not hex"}}

	err := run(t.Context(), cfg, time.Now())
	require.Error(t, err)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "job.RotateKeys", spans[0].Name())
	require.Equal(t, codes.Error, spans[0].Status().Code)
	require.Contains(t, spans[0].Attributes(), otel.ErrorType(err))
}
