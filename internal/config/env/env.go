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

// Raw values for environment variables.
var (
	postgresDsn          = getEnv("POSTGRES_DSN")
	postgresHost         = getEnv("POSTGRES_HOST")
	postgresPort         = getEnv("POSTGRES_PORT")
	postgresUser         = getEnv("POSTGRES_USER")
	postgresPassword     = getEnv("POSTGRES_PASSWORD")
	postgresDatabase     = getEnv("POSTGRES_DATABASE")
	postgresTLSEnabled   = getEnv("POSTGRES_TLS_ENABLED")
	postgresMaxOpenConns = getEnv("POSTGRES_MAX_OPEN_CONNS")
	postgresMaxIdleConns = getEnv("POSTGRES_MAX_IDLE_CONNS")

	appName      = getEnv("APP_NAME")
	appMasterKey = getEnv("APP_MASTER_KEY")
	otel         = getEnv("OTEL")

	grpcUrl = getEnv("GRPC_URL")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")

	downtimeStart = getEnv("DOWNTIME_START")
)

var (
	// PostgresDsn is the legacy URL used when PostgresHost is empty.
	//
	// Configure the discrete PostgreSQL fields for new deployments.
	PostgresDsn = postgresDsn
	// PostgresHost is the PostgreSQL server hostname or IP address.
	PostgresHost = postgresHost
	// PostgresPort is the PostgreSQL server port.
	PostgresPort = config.LoadEnv(postgresPort, PostgresPortDefault, config.IntParser)
	// PostgresUser is the PostgreSQL login role.
	PostgresUser = postgresUser
	// PostgresPassword is the PostgreSQL login credential.
	PostgresPassword = postgresPassword
	// PostgresDatabase is the PostgreSQL database name.
	PostgresDatabase = postgresDatabase
	// PostgresTLSEnabled controls transport encryption for the PostgreSQL connection.
	PostgresTLSEnabled = config.LoadEnv(postgresTLSEnabled, PostgresTLSEnabledDefault, config.BoolParser)

	// PostgresMaxOpenConns is the maximum number of open connections to the database.
	PostgresMaxOpenConns = config.LoadEnv(postgresMaxOpenConns, PostgresMaxOpenConnsDefault, config.IntParser)
	// PostgresMaxIdleConns is the maximum number of connections kept open while idle.
	PostgresMaxIdleConns = config.LoadEnv(postgresMaxIdleConns, PostgresMaxIdleConnsDefault, config.IntParser)

	// AppName is the application name, as it appears in logs and tracing.
	AppName = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	// AppMasterKey is a secure, 32-byte random secret used to encrypt private JSON Web Keys
	// in the database.
	AppMasterKey = appMasterKey
	// Otel configures whether to enable OpenTelemetry tracing.
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	// GrpcUrl is the address of the gRPC service, in the form <host>:<port>.
	GrpcUrl = grpcUrl

	// Grpc configures the gRPC server from the GRPC_* variables.
	Grpc = grpcf.LoadServerConfig(getEnv, grpcf.ServerConfigDefault)
	// Rest configures the REST server from the REST_* variables.
	Rest = httpf.LoadServerConfig(getEnv, restDefaults())

	// GcloudProjectId is the Google Cloud project ID. When set, the service switches to
	// Google Cloud Logging and Google Cloud Trace for observability. When empty, it falls
	// back to local-development logging and disabled tracing.
	GcloudProjectId = gcloudProjectId

	// DowntimeStart is when a planned downtime of this service starts, in RFC 3339. From then
	// until it is removed, the service refuses work. Nil when none is planned.
	DowntimeStart = config.LoadEnv(downtimeStart, nil, downtime.ParseStart)
)

// restDefaults keeps golib's REST defaults but for the versioned ping path and the read-only methods.
func restDefaults() httpf.ServerConfig {
	defaults := httpf.ServerConfigDefault
	defaults.Liveness = "/v2/ping"
	defaults.Cors.AllowedMethods = []string{http.MethodHead, http.MethodGet}

	return defaults
}
