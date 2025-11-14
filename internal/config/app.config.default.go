package config

import (
	"time"

	"github.com/samber/lo"

	"github.com/a-novel/golib/config"
	"github.com/a-novel/golib/logging"
	loggingpresets "github.com/a-novel/golib/logging/presets"
	"github.com/a-novel/golib/otel"
	otelpresets "github.com/a-novel/golib/otel/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

const (
	OtelFlushTimeout = 2 * time.Second
)

var OtelProd = otelpresets.Gcloud{
	ProjectID:    env.GcloudProjectId,
	FlushTimeout: OtelFlushTimeout,
}

var OtelDev = otelpresets.Local{
	FlushTimeout: OtelFlushTimeout,
}

var NoOtel = otelpresets.Disabled{}

var LoggerProd = loggingpresets.GrpcGcloud{
	Component: env.GcloudProjectId,
}

var LoggerDev = loggingpresets.GrpcLocal{}

var AppPresetDefault = App{
	App: Main{
		Name:      config.LoadEnv(env.AppName, env.AppNameDefault, config.StringParser),
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
