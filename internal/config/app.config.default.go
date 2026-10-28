package config

import (
	"os"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"
	"github.com/a-novel-kit/golib/sentryf"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

// OtelFlushTimeout bounds how long the OpenTelemetry exporter waits to flush pending spans on
// shutdown.
const OtelFlushTimeout = 2 * time.Second

// SentryFlushTimeout bounds how long a server waits on exit to send its pending Sentry events, inside
// the platform's shutdown grace period.
const SentryFlushTimeout = 2 * time.Second

// SentryJobFlushTimeout bounds a job's wait on exit for its pending Sentry events. It outlasts
// sentryf.SendTimeout, so an event sent before the network came up still leaves.
const SentryJobFlushTimeout = sentryf.SendTimeout + time.Minute

// OtelJobFlushTimeout bounds a job's wait on exit for its pending spans and logs. A job often ends
// before the network lets its first connection out. The exporter then sends the spans, then the
// logs, within 30 seconds each.
const OtelJobFlushTimeout = time.Minute

// OtelPresetDefault is the OpenTelemetry exporter of the servers. It exports nothing unless OTEL is
// set, then exports over OTLP/HTTP when OTEL_EXPORTER_OTLP_ENDPOINT is set, and to standard output
// otherwise. The liveness probe produces no span.
var OtelPresetDefault = newOtelPreset(OtelFlushTimeout)

// OtelJobPresetDefault is the OpenTelemetry exporter of the rotation job. It exports as
// [OtelPresetDefault] does, and flushes within [OtelJobFlushTimeout].
var OtelJobPresetDefault = newOtelPreset(OtelJobFlushTimeout)

// exportsOTLP reports whether the servers export over OTLP: OTEL is set, and so is the OTLP
// endpoint. Their logs then go through the same backend, and their spans record each request.
var exportsOTLP = env.Otel && env.OTLPEndpoint != ""

// LoggerDevGrpc prints gRPC request logs to the console for local development.
var LoggerDevGrpc = loggingpresets.GRPCLocal{}

// LoggerDev prints application and REST request logs to the console for local development.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd sends application logs through the OpenTelemetry backend, linked to their trace.
var LoggerProd = &loggingpresets.LogOTel{}

// AppPresetDefault is the configuration the servers start with. It reads every value from the
// environment, and logs through the OpenTelemetry backend when [exportsOTLP] holds. Spans then
// record each request and call, so the servers write no request log.
var AppPresetDefault = App{
	App: Main{
		Name:          env.AppName,
		MasterKey:     env.AppMasterKey,
		DowntimeStart: env.DowntimeStart,
	},
	Grpc: env.Grpc,
	Rest: env.Rest,

	Otel:       OtelPresetDefault,
	Sentry:     env.Sentry,
	Logger:     lo.Ternary[logging.Log](!exportsOTLP, LoggerDev, LoggerProd),
	GrpcLogger: lo.Ternary[logging.RPCConfig](!exportsOTLP, &LoggerDevGrpc, &loggingpresets.GRPCOTel{}),
	RestLogger: lo.Ternary[logging.HTTPConfig](
		!exportsOTLP, &loggingpresets.HTTPLocal{BaseLogger: LoggerDev}, nil,
	),
	Postgres: PostgresPresetDefault,
}

// newOtelPreset picks the exporter [OtelPresetDefault] describes, flushing within flushTimeout.
func newOtelPreset(flushTimeout time.Duration) otel.Config {
	switch {
	case !env.Otel:
		return &otelpresets.Disabled{}
	case env.OTLPEndpoint == "":
		return &otelpresets.Local{FlushTimeout: flushTimeout}
	default:
		return &otelpresets.OTLP{UntracedPaths: []string{env.Rest.Liveness}, FlushTimeout: flushTimeout}
	}
}
