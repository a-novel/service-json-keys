package config

import (
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"

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

// LoggerProdGrpc sends production-ready logs to Google Cloud environment.
var LoggerProdGrpc = loggingpresets.GrpcGcloud{
	Component: env.GcloudProjectId,
}

// LoggerDevGrpc prints logs in the console, pretty formatted.
var LoggerDevGrpc = loggingpresets.GrpcLocal{}

// LoggerDevHttp prints HTTP-level logs in the console, pretty formatted.
var LoggerDevHttp = &loggingpresets.LogLocal{
	Out:      os.Stdout,
	Renderer: lipgloss.NewRenderer(os.Stdout, termenv.WithTTY(true)),
}

// LoggerProdHttp sends HTTP-level production-ready logs to Google Cloud environment.
var LoggerProdHttp = &loggingpresets.LogGcloud{
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

	Otel: lo.If[otel.Config](!env.Otel, &NoOtel).
		ElseIf(env.GcloudProjectId == "", &OtelDev).
		Else(&OtelProd),
	GrpcLogger: lo.Ternary[logging.RpcConfig](env.GcloudProjectId == "", &LoggerDevGrpc, &LoggerProdGrpc),
	HttpLogger: lo.Ternary[logging.HttpConfig](
		env.GcloudProjectId == "",
		&loggingpresets.HttpLocal{BaseLogger: LoggerDevHttp},
		&loggingpresets.HttpGcloud{BaseLogger: LoggerProdHttp},
	),
	Postgres: PostgresPresetDefault,
}
