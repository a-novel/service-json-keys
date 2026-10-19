package config

import (
	"os"
	"time"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

const (
	// OtelFlushTimeout is the maximum time allowed for the OpenTelemetry exporter to
	// flush pending spans on shutdown.
	OtelFlushTimeout = 2 * time.Second
)

// OtelPresetDefault is the OpenTelemetry exporter of the servers and the rotation job. It exports
// nothing unless OTEL is set, then exports to Google Cloud when GCLOUD_PROJECT_ID is set, and to
// standard output otherwise.
var OtelPresetDefault = lo.If[otel.Config](!env.Otel, &otelpresets.Disabled{}).
	ElseIf(env.GcloudProjectId == "", &otelpresets.Local{
		FlushTimeout: OtelFlushTimeout,
	}).
	Else(&otelpresets.Gcloud{
		ProjectID:    env.GcloudProjectId,
		FlushTimeout: OtelFlushTimeout,
	})

// LoggerProdGrpc sends gRPC request logs to Google Cloud Logging.
var LoggerProdGrpc = loggingpresets.GRPCGcloud{
	Component: env.AppName,
	ProjectId: env.GcloudProjectId,
}

// LoggerDevGrpc prints gRPC request logs to the console in a human-readable format.
var LoggerDevGrpc = loggingpresets.GRPCLocal{}

// LoggerDev prints HTTP request logs to the console in a human-readable format.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd sends HTTP request logs to Google Cloud Logging.
var LoggerProd = &loggingpresets.LogGcloud{
	ProjectId: env.GcloudProjectId,
}

// AppPresetDefault is the default [App] configuration populated from environment variables.
var AppPresetDefault = App{
	App: Main{
		Name:          env.AppName,
		MasterKey:     env.AppMasterKey,
		DowntimeStart: env.DowntimeStart,
	},
	Grpc: env.Grpc,
	Rest: env.Rest,

	Otel:       OtelPresetDefault,
	Logger:     lo.Ternary[logging.Log](env.GcloudProjectId == "", LoggerDev, LoggerProd),
	GrpcLogger: lo.Ternary[logging.RPCConfig](env.GcloudProjectId == "", &LoggerDevGrpc, &LoggerProdGrpc),
	RestLogger: lo.Ternary[logging.HTTPConfig](
		env.GcloudProjectId == "",
		&loggingpresets.HTTPLocal{BaseLogger: LoggerDev},
		&loggingpresets.HTTPGcloud{BaseLogger: LoggerProd},
	),
	Postgres: PostgresPresetDefault,
}
