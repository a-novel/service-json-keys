package env

import (
	"net/http"
	"os"

	"github.com/a-novel-kit/golib/config"
	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
)

// prefix is the value of SERVICE_JSON_KEYS_ENV_PREFIX, prepended to all environment
// variable names read by this package. Set it to avoid conflicts when multiple instances
// of this service run in the same environment.
var prefix = os.Getenv("SERVICE_JSON_KEYS_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// Default values used when the corresponding environment variable is absent.
const (
	AppNameDefault = "service-json-keys"

	// PostgresMaxOpenConnsDefault keeps the pool well under a stock PostgreSQL
	// max_connections of 100 once multiplied by a service's replica count, leaving
	// room for the migration job and a psql session. Go's own default is unlimited,
	// which turns a spike into connection refusals for everything on that database
	// rather than queueing inside this process.
	PostgresMaxOpenConnsDefault = 20
	// PostgresMaxIdleConnsDefault matches the open limit so a burst does not close
	// connections it is about to reopen.
	PostgresMaxIdleConnsDefault = 20
	PostgresPortDefault         = 5432
	PostgresTLSEnabledDefault   = true
)

var (
	// PostgresDsn is the legacy URL used when PostgresHost is empty.
	//
	// Configure the discrete PostgreSQL fields for new deployments.
	PostgresDsn = getEnv("POSTGRES_DSN")
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

	// GrpcUrl is the address of the gRPC service, in the form <host>:<port>.
	GrpcUrl = getEnv("GRPC_URL")

	// Grpc configures the gRPC server from the GRPC_* variables.
	Grpc = grpcf.LoadServerConfig(getEnv, grpcf.ServerConfigDefault)
	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())

	// GcloudProjectId is the Google Cloud project ID. When set, logs and traces go to Google
	// Cloud. When empty, they go to standard output.
	GcloudProjectId = getEnv("GCLOUD_PROJECT_ID")

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
