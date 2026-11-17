package env

import (
	"cmp"
	"net/http"
	"os"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	otelpresets "github.com/a-novel-kit/golib/otel/presets"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"
	"github.com/a-novel-kit/golib/servicetoken"
)

// prefix is prepended to every configuration variable name that this package reads. Set
// SERVICE_JSON_KEYS_ENV_PREFIX when several instances of the service share one environment.
var prefix = os.Getenv("SERVICE_JSON_KEYS_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// AppNameDefault names the application in logs and traces when APP_NAME is unset.
const AppNameDefault = "service-json-keys"

var (
	// Postgres reaches PostgreSQL through the POSTGRES_* variables.
	Postgres = postgrespresets.LoadConnection(getEnv, postgrespresets.ConnectionDefault)

	// AppName is the application name, as it appears in logs and tracing.
	AppName = cmp.Or(getEnv("APP_NAME"), AppNameDefault)
	// AppMasterKey is the hex-encoded 32-byte secret that encrypts private JSON Web Keys in the
	// database.
	AppMasterKey = getEnv("APP_MASTER_KEY")

	// Telemetry is the servers' OpenTelemetry exporter, which OTEL and OTEL_EXPORTER_OTLP_ENDPOINT
	// pick, with the loggers that match it.
	Telemetry = otelpresets.LoadTelemetry(getEnv, otelpresets.ServerFlushTimeout)
	// JobTelemetry is the rotation job's exporter, picked the same way. It waits longer for its last
	// spans and logs on exit.
	JobTelemetry = otelpresets.LoadTelemetry(getEnv, otelpresets.JobFlushTimeout)

	// Grpc configures the gRPC server from the GRPC_* variables.
	Grpc = grpcf.LoadServerConfig(getEnv, grpcf.ServerConfigDefault)
	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())

	// ServiceToken names the service tokens the gRPC server accepts, from the SERVICE_TOKEN_*
	// variables. Without an issuer, the server refuses to start unless SERVICE_ISSUER_OPTIONAL is set.
	ServiceToken = servicetoken.LoadConfig(getEnv)

	// DowntimeStart is when a planned downtime of this service starts, in RFC 3339. From then
	// until it is removed, the service refuses work. Nil when none is planned.
	DowntimeStart = config.LoadEnv(getEnv("DOWNTIME_START"), nil, downtime.ParseStart)
)

// restDefaults keeps golib's REST defaults but for the versioned ping path and the read-only methods.
func restDefaults() httpf.ServerConfig {
	defaults := httpf.ServerConfigDefault
	defaults.Liveness = "/v2/ping"
	defaults.Cors.AllowedMethods = []string{http.MethodHead, http.MethodGet}

	return defaults
}
