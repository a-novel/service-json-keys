package config

import (
	"time"

	"github.com/samber/lo"

	"github.com/a-novel/golib/logging"
	loggingpresets "github.com/a-novel/golib/logging/presets"
	"github.com/a-novel/golib/otel"
	otelpresets "github.com/a-novel/golib/otel/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

const OtelFlushTimeout = 2 * time.Second

// OtelProd is the production configuration for Open Telemetry. Requires a Google Cloud project.
var OtelProd = otelpresets.Gcloud{
	ProjectID:    env.GcloudProjectId,
	FlushTimeout: OtelFlushTimeout,
}

// OtelDev logs all traces locally.
var OtelDev = otelpresets.Local{
	FlushTimeout: OtelFlushTimeout,
}

// NoOtel runs a disabled otel instance.
var NoOtel = otelpresets.Disabled{}

// LoggerProd sends production-ready logs to Google Cloud environment.
var LoggerProd = loggingpresets.GrpcGcloud{
	Component: env.GcloudProjectId,
}

// LoggerDev prints logs in the console, pretty formatted.
var LoggerDev = loggingpresets.GrpcLocal{}

var AppPresetDefault = App{
	App: Main{
		Name:      env.AppName,
		MasterKey: env.AppMasterKey,
	},
	Grpc: Grpc{
		Port: env.GrpcPort,
		Ping: env.GrpcPing,
	},

	Otel: lo.If[otel.Config](!env.Otel, &NoOtel).
		ElseIf(env.GcloudProjectId == "", &OtelDev).
		Else(&OtelProd),
	Logger:   lo.Ternary[logging.RpcConfig](env.GcloudProjectId == "", &LoggerDev, &LoggerProd),
	Postgres: PostgresPresetDefault,
}
