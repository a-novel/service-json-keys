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
	OtelFlushTimeout = 2 * time.Second
)

// LoggerProdGrpc sends production-ready logs to Google Cloud environment.
var LoggerProdGrpc = loggingpresets.GrpcGcloud{
	Component: env.GcloudProjectId,
}

// LoggerDevGrpc prints logs in the console, pretty formatted.
var LoggerDevGrpc = loggingpresets.GrpcLocal{}

// LoggerDev prints HTTP-level logs in the console, pretty formatted.
var LoggerDev = &loggingpresets.LogLocal{
	Out: os.Stdout,
}

// LoggerProd sends HTTP-level production-ready logs to Google Cloud environment.
var LoggerProd = &loggingpresets.LogGcloud{
	ProjectId: env.GcloudProjectId,
}

var AppPresetDefault = App{
	App: Main{
		Name:      env.AppName,
		MasterKey: env.AppMasterKey,
	},
	Grpc: Grpc{
		Port: env.GrpcPort,
		Ping: env.GrpcPing,
	},
	Rest: Rest{
		Port: env.RestPort,
		Timeouts: RestTimeouts{
			Read:       env.RestTimeoutRead,
			ReadHeader: env.RestTimeoutReadHeader,
			Write:      env.RestTimeoutWrite,
			Idle:       env.RestTimeoutIdle,
			Request:    env.RestTimeoutRequest,
		},
		MaxRequestSize: env.RestMaxRequestSize,
		Cors: RestCors{
			AllowedOrigins:   env.CorsAllowedOrigins,
			AllowedHeaders:   env.CorsAllowedHeaders,
			AllowCredentials: env.CorsAllowCredentials,
			MaxAge:           env.CorsMaxAge,
		},
	},

	Otel: lo.If[otel.Config](!env.Otel, &otelpresets.Disabled{}).
		ElseIf(env.GcloudProjectId == "", &otelpresets.Local{
			FlushTimeout: OtelFlushTimeout,
		}).
		Else(&otelpresets.Gcloud{
			ProjectID:    env.GcloudProjectId,
			FlushTimeout: OtelFlushTimeout,
		}),
	Logger:     lo.Ternary[logging.Log](env.GcloudProjectId == "", LoggerDev, LoggerProd),
	GrpcLogger: lo.Ternary[logging.RpcConfig](env.GcloudProjectId == "", &LoggerDevGrpc, &LoggerProdGrpc),
	HttpLogger: lo.Ternary[logging.HttpConfig](
		env.GcloudProjectId == "",
		&loggingpresets.HttpLocal{BaseLogger: LoggerDev},
		&loggingpresets.HttpGcloud{BaseLogger: LoggerProd},
	),
	Postgres: PostgresPresetDefault,
}
