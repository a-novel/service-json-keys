package env

import (
	"os"
	"time"

	"github.com/a-novel-kit/golib/config"
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

	GrpcPortDefault = 8080
	GrpcDefaultPing = time.Second * 5

	RestPortDefault              = 8080
	RestTimeoutReadDefault       = 15 * time.Second
	RestTimeoutReadHeaderDefault = 3 * time.Second
	RestTimeoutWriteDefault      = 30 * time.Second
	RestTimeoutIdleDefault       = 60 * time.Second
	RestTimeoutRequestDefault    = 60 * time.Second
	RestMaxRequestSizeDefault    = 2 << 20 // 2 MiB
	CorsAllowCredentialsDefault  = false
	CorsMaxAgeDefault            = 3600
)

// Default values used when the corresponding environment variable is absent.
var (
	CorsAllowedOriginsDefault = []string{"*"}
	CorsAllowedHeadersDefault = []string{"*"}
)

// Raw values for environment variables.
var (
	postgresDsn = getEnv("POSTGRES_DSN")

	appName      = getEnv("APP_NAME")
	appMasterKey = getEnv("APP_MASTER_KEY")
	otel         = getEnv("OTEL")

	grpcPort = getEnv("GRPC_PORT")
	grpcUrl  = getEnv("GRPC_URL")
	grpcPing = getEnv("GRPC_PING")

	restPort              = getEnv("REST_PORT")
	restTimeoutRead       = getEnv("REST_TIMEOUT_READ")
	restTimeoutReadHeader = getEnv("REST_TIMEOUT_READ_HEADER")
	restTimeoutWrite      = getEnv("REST_TIMEOUT_WRITE")
	restTimeoutIdle       = getEnv("REST_TIMEOUT_IDLE")
	restTimeoutRequest    = getEnv("REST_TIMEOUT_REQUEST")
	restMaxRequestSize    = getEnv("REST_MAX_REQUEST_SIZE")

	corsAllowedOrigins   = getEnv("REST_CORS_ALLOWED_ORIGINS")
	corsAllowedHeaders   = getEnv("REST_CORS_ALLOWED_HEADERS")
	corsAllowCredentials = getEnv("REST_CORS_ALLOW_CREDENTIALS")
	corsMaxAge           = getEnv("REST_CORS_MAX_AGE")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")
)

var (
	// PostgresDsn is the URL used to connect to the Postgres database instance:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsn = postgresDsn

	// AppName is the application name, as it appears in logs and tracing.
	AppName = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	// AppMasterKey is a secure, 32-byte random secret used to encrypt private JSON Web Keys
	// in the database.
	AppMasterKey = appMasterKey
	// Otel configures whether to enable OpenTelemetry tracing.
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	// GrpcPort is the port on which the gRPC server listens for incoming requests.
	GrpcPort = config.LoadEnv(grpcPort, GrpcPortDefault, config.IntParser)
	// GrpcUrl is the address of the gRPC service, in the form <host>:<port>.
	GrpcUrl = grpcUrl
	// GrpcPing configures the refresh interval for the gRPC server internal healthcheck.
	GrpcPing = config.LoadEnv(grpcPing, GrpcDefaultPing, config.DurationParser)

	// RestPort is the port on which the REST server listens for incoming requests.
	RestPort = config.LoadEnv(restPort, RestPortDefault, config.IntParser)
	// RestTimeoutRead is the maximum duration for reading an incoming REST request.
	RestTimeoutRead = config.LoadEnv(restTimeoutRead, RestTimeoutReadDefault, config.DurationParser)
	// RestTimeoutReadHeader is the maximum duration for reading the headers of an incoming REST request.
	RestTimeoutReadHeader = config.LoadEnv(restTimeoutReadHeader, RestTimeoutReadHeaderDefault, config.DurationParser)
	// RestTimeoutWrite is the maximum duration for writing a REST response.
	RestTimeoutWrite = config.LoadEnv(restTimeoutWrite, RestTimeoutWriteDefault, config.DurationParser)
	// RestTimeoutIdle is the maximum duration to wait for the next request when keep-alives are enabled.
	RestTimeoutIdle = config.LoadEnv(restTimeoutIdle, RestTimeoutIdleDefault, config.DurationParser)
	// RestTimeoutRequest is the maximum duration for processing an incoming REST request.
	RestTimeoutRequest = config.LoadEnv(restTimeoutRequest, RestTimeoutRequestDefault, config.DurationParser)
	// RestMaxRequestSize is the maximum size of an incoming REST request body.
	RestMaxRequestSize = config.LoadEnv(restMaxRequestSize, RestMaxRequestSizeDefault, config.Int64Parser)

	// CorsAllowedOrigins lists the origins allowed to access the REST API.
	CorsAllowedOrigins = config.LoadEnv(
		corsAllowedOrigins, CorsAllowedOriginsDefault, config.SliceParser(config.StringParser),
	)
	// CorsAllowedHeaders lists the headers allowed in CORS requests.
	CorsAllowedHeaders = config.LoadEnv(
		corsAllowedHeaders, CorsAllowedHeadersDefault, config.SliceParser(config.StringParser),
	)
	// CorsAllowCredentials configures whether CORS requests can include credentials.
	CorsAllowCredentials = config.LoadEnv(corsAllowCredentials, CorsAllowCredentialsDefault, config.BoolParser)
	// CorsMaxAge is the maximum age, in seconds, of CORS preflight cache results.
	CorsMaxAge = config.LoadEnv(corsMaxAge, CorsMaxAgeDefault, config.IntParser)

	// GcloudProjectId is the Google Cloud project ID. When set, the service switches to
	// Google Cloud Logging and Google Cloud Trace for observability. When empty, it falls
	// back to local-development logging and disabled tracing.
	GcloudProjectId = gcloudProjectId
)
