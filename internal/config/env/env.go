package env

import (
	"os"
	"time"

	"github.com/a-novel-kit/golib/config"
)

// Prefix allows to set a custom prefix to all configuration environment variables.
// This is useful when importing the package in another project, when env variable names
// might conflict with the source project.
var prefix = os.Getenv("SERVICE_JSON_KEYS_ENV_PREFIX")

func getEnv(name string) string {
	return os.Getenv(prefix + name)
}

// Default values for environment variables, if applicable.
const (
	appNameDefault = "service-authentication"

	grpcPortDefault = 8080
	grpcDefaultPing = time.Second * 5
)

// Raw values for environment variables.
var (
	postgresDsn     = getEnv("POSTGRES_DSN")
	postgresDsnTest = getEnv("POSTGRES_DSN_TEST")

	appName      = getEnv("APP_NAME")
	appMasterKey = getEnv("APP_MASTER_KEY")
	otel         = getEnv("OTEL")

	grpcPort     = getEnv("GRPC_PORT")
	grpcTestPort = getEnv("GRPC_TEST_PORT")
	grpcUrl      = getEnv("GRPC_URL")
	grpcTestUrl  = getEnv("GRPC_TEST_URL")
	grpcPing     = getEnv("GRPC_PING")

	gcloudProjectId = getEnv("GCLOUD_PROJECT_ID")
)

var (
	// PostgresDsn is the url used to connect to the postgres database instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsn = postgresDsn
	// PostgresDsnTest is the url used to connect to the postgres database test instance.
	// Typically formatted as:
	//	postgres://<user>:<password>@<host>:<port>/<database>
	PostgresDsnTest = postgresDsnTest

	// AppName is the name of the application, as it will appear in logs and tracing.
	AppName = config.LoadEnv(appName, appNameDefault, config.StringParser)
	// AppMasterKey is a secure, 32-byte random secret used to encrypt private JSON keys
	// in the database.
	AppMasterKey = appMasterKey
	// Otel flag configures whether to use Open Telemetry or not.
	//
	// See: https://opentelemetry.io/
	Otel = config.LoadEnv(otel, false, config.BoolParser)

	// GrpcPort is the port on which the Grpc server will listen for incoming requests.
	GrpcPort = config.LoadEnv(grpcPort, grpcPortDefault, config.IntParser)
	// GrpcUrl is the url of the Grpc service, typically <host>:<port>.
	GrpcUrl = grpcUrl
	// GrpcTestPort is the port on which the Grpc test server will listen for incoming requests.
	GrpcTestPort = config.LoadEnv(grpcTestPort, grpcPortDefault, config.IntParser)
	// GrpcTestUrl is the url of the Grpc test service, typically <host>:<port>.
	GrpcTestUrl = grpcTestUrl
	// GrpcPing configures the refresh interval for the Grpc server internal healthcheck.
	GrpcPing = config.LoadEnv(grpcPing, grpcDefaultPing, config.DurationParser)

	// GcloudProjectId configures the server for Google Cloud environment.
	//
	// See: https://docs.cloud.google.com/resource-manager/docs/creating-managing-projects
	GcloudProjectId = gcloudProjectId
)
