package env

import (
	"net/http"
	"os"
	"time"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
)

// prefix is prepended to every configuration variable name that this package reads. Set
// SERVICE_JSON_KEYS_ENV_PREFIX when several instances of the service share one environment.
var prefix = os.Getenv("SERVICE_JSON_KEYS_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

const (
	// AppNameDefault names the application in logs and traces when APP_NAME is unset.
	AppNameDefault = "service-json-keys"

	// TimeoutShutdownDefault ends a server's drain 6 seconds after SIGTERM, so its OpenTelemetry
	// flush, up to 2 seconds, finishes long before Kubernetes kills the pod at 30 seconds.
	TimeoutShutdownDefault = 6 * time.Second

	// PostgresMaxOpenConnsDefault caps the connections one instance opens. Multiplied by the replica
	// count, the cap stays well under PostgreSQL's default max_connections of 100. That leaves room
	// for the migration job and a psql session. A spike waits inside the process for a free connection.
	PostgresMaxOpenConnsDefault = 20
	// PostgresMaxIdleConnsDefault matches the open limit, so a burst keeps the connections it is about
	// to reuse.
	PostgresMaxIdleConnsDefault = 20
	// PostgresPortDefault is PostgreSQL's standard port.
	PostgresPortDefault = 5432
	// PostgresTLSEnabledDefault encrypts the connection unless a deployment turns it off.
	PostgresTLSEnabledDefault = true
)

var (
	// PostgresHost is the PostgreSQL server hostname or IP address.
	PostgresHost = getEnv("POSTGRES_HOST")
	// PostgresPort is the PostgreSQL server port.
	PostgresPort = config.LoadEnv(getEnv("POSTGRES_PORT"), PostgresPortDefault, config.IntParser)
	// PostgresUser is the PostgreSQL login role.
	PostgresUser = getEnv("POSTGRES_USER")
	// PostgresPassword is the PostgreSQL login credential.
	PostgresPassword = getEnv("POSTGRES_PASSWORD")
	// PostgresDatabase is the PostgreSQL database name.
	PostgresDatabase = getEnv("POSTGRES_DATABASE")
	// PostgresTLSEnabled controls transport encryption for the PostgreSQL connection.
	PostgresTLSEnabled = config.LoadEnv(getEnv("POSTGRES_TLS_ENABLED"), PostgresTLSEnabledDefault, config.BoolParser)

	// PostgresMaxOpenConns is the maximum number of open connections to the database.
	PostgresMaxOpenConns = config.LoadEnv(getEnv("POSTGRES_MAX_OPEN_CONNS"), PostgresMaxOpenConnsDefault, config.IntParser)
	// PostgresMaxIdleConns is the maximum number of connections kept open while idle.
	PostgresMaxIdleConns = config.LoadEnv(getEnv("POSTGRES_MAX_IDLE_CONNS"), PostgresMaxIdleConnsDefault, config.IntParser)

	// AppName is the application name, as it appears in logs and tracing.
	AppName = config.LoadEnv(getEnv("APP_NAME"), AppNameDefault, config.StringParser)
	// AppMasterKey is the hex-encoded 32-byte secret that encrypts private JSON Web Keys in the
	// database.
	AppMasterKey = getEnv("APP_MASTER_KEY")
	// Otel enables OpenTelemetry tracing.
	Otel = config.LoadEnv(getEnv("OTEL"), false, config.BoolParser)

	// GrpcURL is the address of the gRPC service, in the form <host>:<port>.
	GrpcURL = getEnv("GRPC_URL")

	// Grpc configures the gRPC server from the GRPC_* variables.
	Grpc = grpcf.LoadServerConfig(getEnv, grpcDefaults())
	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())

	// ServiceTokenIssuer is the issuer of the service tokens the gRPC server accepts, as a token's
	// "iss" claim names it. When empty, the server takes every call without a service token.
	ServiceTokenIssuer = getEnv("SERVICE_TOKEN_ISSUER")
	// ServiceTokenAudience is the audience the gRPC server answers to. Each caller requests its token
	// for it.
	ServiceTokenAudience = getEnv("SERVICE_TOKEN_AUDIENCE")
	// ServiceTokenCallers are the service accounts allowed to call, as a comma-separated list of
	// system:serviceaccount:<namespace>:<name>.
	ServiceTokenCallers = config.LoadEnv(getEnv("SERVICE_TOKEN_CALLERS"), nil, config.SliceParser(config.StringParser))

	// OTLPEndpoint is the standard OTEL_EXPORTER_OTLP_ENDPOINT. When set, traces and logs export
	// over OTLP/HTTP, which the OTEL_EXPORTER_OTLP_* variables configure. When empty, they go to
	// standard output. The OpenTelemetry SDK reads these variables without the prefix, so this one
	// is read without it too.
	OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	// DowntimeStart is when a planned downtime of this service starts, in RFC 3339. From then
	// until it is removed, the service refuses work. Nil when none is planned.
	DowntimeStart = config.LoadEnv(getEnv("DOWNTIME_START"), nil, downtime.ParseStart)
)

// restDefaults keeps golib's REST defaults but for the versioned ping path and the read-only methods.
func restDefaults() httpf.ServerConfig {
	defaults := httpf.ServerConfigDefault
	defaults.Liveness = "/v2/ping"
	defaults.Timeouts.Shutdown = TimeoutShutdownDefault
	defaults.Cors.AllowedMethods = []string{http.MethodHead, http.MethodGet}

	return defaults
}

// grpcDefaults keeps golib's gRPC defaults but for the drain, which fits the pod's grace period.
func grpcDefaults() grpcf.ServerConfig {
	defaults := grpcf.ServerConfigDefault
	defaults.Shutdown = TimeoutShutdownDefault

	return defaults
}
