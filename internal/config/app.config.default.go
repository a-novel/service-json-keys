package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/samber/lo"

	configparser "github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/logging"
	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/a-novel-kit/golib/otel"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

const (
	// OtelFlushTimeout bounds exporter shutdown.
	OtelFlushTimeout = 2 * time.Second
)

// LoggerDev writes human-readable logs to standard output.
var LoggerDev = &loggingpresets.LogLocal{Out: os.Stdout}

// LoadApp reads and validates the runtime configuration for the gRPC and REST servers.
func LoadApp() (App, error) {
	var (
		appName = env.AppNameDefault

		grpcPort     = env.GrpcPortDefault
		grpcPing     = env.GrpcDefaultPing
		grpcShutdown = env.GrpcTimeoutShutdownDefault

		restPort        = env.RestPortDefault
		restRead        = env.RestTimeoutReadDefault
		restReadHeader  = env.RestTimeoutReadHeaderDefault
		restWrite       = env.RestTimeoutWriteDefault
		restIdle        = env.RestTimeoutIdleDefault
		restRequest     = env.RestTimeoutRequestDefault
		restShutdown    = env.RestTimeoutShutdownDefault
		restRequestSize = int64(env.RestMaxRequestSizeDefault)

		corsOrigins     = env.CorsAllowedOriginsDefault
		corsHeaders     = env.CorsAllowedHeadersDefault
		corsCredentials = env.CorsAllowCredentialsDefault
		corsMaxAge      = env.CorsMaxAgeDefault

		otelEnabled bool
		loadErrors  []error
	)

	loadValue(&loadErrors, &appName, "APP_NAME", env.AppNameDefault, configparser.StringParser)
	loadValue(&loadErrors, &grpcPort, "GRPC_PORT", env.GrpcPortDefault, configparser.IntParser)
	loadValue(&loadErrors, &grpcPing, "GRPC_PING", env.GrpcDefaultPing, configparser.DurationParser)
	loadValue(
		&loadErrors,
		&grpcShutdown,
		"GRPC_TIMEOUT_SHUTDOWN",
		env.GrpcTimeoutShutdownDefault,
		configparser.DurationParser,
	)
	loadValue(&loadErrors, &restPort, "REST_PORT", env.RestPortDefault, configparser.IntParser)
	loadValue(
		&loadErrors,
		&restRead,
		"REST_TIMEOUT_READ",
		env.RestTimeoutReadDefault,
		configparser.DurationParser,
	)
	loadValue(
		&loadErrors,
		&restReadHeader,
		"REST_TIMEOUT_READ_HEADER",
		env.RestTimeoutReadHeaderDefault,
		configparser.DurationParser,
	)
	loadValue(
		&loadErrors,
		&restWrite,
		"REST_TIMEOUT_WRITE",
		env.RestTimeoutWriteDefault,
		configparser.DurationParser,
	)
	loadValue(
		&loadErrors,
		&restIdle,
		"REST_TIMEOUT_IDLE",
		env.RestTimeoutIdleDefault,
		configparser.DurationParser,
	)
	loadValue(
		&loadErrors,
		&restRequest,
		"REST_TIMEOUT_REQUEST",
		env.RestTimeoutRequestDefault,
		configparser.DurationParser,
	)
	loadValue(
		&loadErrors,
		&restShutdown,
		"REST_TIMEOUT_SHUTDOWN",
		env.RestTimeoutShutdownDefault,
		configparser.DurationParser,
	)
	loadValue(
		&loadErrors,
		&restRequestSize,
		"REST_MAX_REQUEST_SIZE",
		int64(env.RestMaxRequestSizeDefault),
		configparser.Int64Parser,
	)
	loadValue(
		&loadErrors,
		&corsOrigins,
		"REST_CORS_ALLOWED_ORIGINS",
		env.CorsAllowedOriginsDefault,
		configparser.SliceParser(configparser.StringParser),
	)
	loadValue(
		&loadErrors,
		&corsHeaders,
		"REST_CORS_ALLOWED_HEADERS",
		env.CorsAllowedHeadersDefault,
		configparser.SliceParser(configparser.StringParser),
	)
	loadValue(
		&loadErrors,
		&corsCredentials,
		"REST_CORS_ALLOW_CREDENTIALS",
		env.CorsAllowCredentialsDefault,
		configparser.BoolParser,
	)
	loadValue(&loadErrors, &corsMaxAge, "REST_CORS_MAX_AGE", env.CorsMaxAgeDefault, configparser.IntParser)
	loadValue(&loadErrors, &otelEnabled, "OTEL", false, configparser.BoolParser)

	postgresConfig, err := LoadPostgres()
	if err != nil {
		loadErrors = append(loadErrors, err)
	}

	err = errors.Join(loadErrors...)
	if err != nil {
		return App{}, fmt.Errorf("load application config: %w", err)
	}

	gcloudProjectID := env.Get("GCLOUD_PROJECT_ID")
	loggerProd := &loggingpresets.LogGcloud{ProjectId: gcloudProjectID}

	otelConfig := lo.If[otel.Config](!otelEnabled, &otelpresets.Disabled{}).
		ElseIf(gcloudProjectID == "", &otelpresets.Local{FlushTimeout: OtelFlushTimeout}).
		Else(&otelpresets.Gcloud{ProjectID: gcloudProjectID, FlushTimeout: OtelFlushTimeout})

	return App{
		App: Main{
			Name:      appName,
			MasterKey: env.Get("APP_MASTER_KEY"),
		},
		Grpc: Grpc{
			Port:     grpcPort,
			Ping:     grpcPing,
			Shutdown: grpcShutdown,
		},
		Rest: Rest{
			Port: restPort,
			Timeouts: RestTimeouts{
				Read:       restRead,
				ReadHeader: restReadHeader,
				Write:      restWrite,
				Idle:       restIdle,
				Request:    restRequest,
				Shutdown:   restShutdown,
			},
			MaxRequestSize: restRequestSize,
			Cors: RestCors{
				AllowedOrigins:   corsOrigins,
				AllowedHeaders:   corsHeaders,
				AllowCredentials: corsCredentials,
				MaxAge:           corsMaxAge,
			},
		},
		Otel:            otelConfig,
		Logger:          lo.Ternary[logging.Log](gcloudProjectID == "", LoggerDev, loggerProd),
		GrpcLogger:      grpcLogger(gcloudProjectID),
		RestLogger:      restLogger(gcloudProjectID, loggerProd),
		Postgres:        postgresConfig,
		GcloudProjectID: gcloudProjectID,
	}, nil
}

func grpcLogger(gcloudProjectID string) logging.RPCConfig {
	if gcloudProjectID == "" {
		return &loggingpresets.GRPCLocal{}
	}

	return &loggingpresets.GRPCGcloud{Component: gcloudProjectID}
}

func restLogger(gcloudProjectID string, loggerProd *loggingpresets.LogGcloud) logging.HTTPConfig {
	if gcloudProjectID == "" {
		return &loggingpresets.HTTPLocal{BaseLogger: LoggerDev}
	}

	return &loggingpresets.HTTPGcloud{BaseLogger: loggerProd}
}

func loadValue[T any](
	loadErrors *[]error,
	target *T,
	name string,
	fallback T,
	parser func(string) (T, error),
) {
	value, err := env.Load(name, fallback, parser)
	if err != nil {
		*loadErrors = append(*loadErrors, err)

		return
	}

	*target = value
}
