package env

import (
	"os"
	"time"

	"github.com/a-novel/golib/config"
)

// Prefix allows to set a custom prefix to all configuration environment variables.
// This is useful when importing the package in another project, when env variable names
// might conflict with the source project.
var Prefix = os.Getenv("SERVICE_JSON_KEYS_ENV_PREFIX")

func getEnv(name string) string {
	if Prefix != "" {
		return os.Getenv(Prefix + "_" + name)
	}

	return os.Getenv(name)
}

const (
	AppNameDefault = "service-authentication"

	grpcPortDefault = 8080
	grpcDefaultPing = time.Second * 5
)

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
	PostgresDsn     = postgresDsn
	PostgresDsnTest = postgresDsnTest

	AppName      = config.LoadEnv(appName, AppNameDefault, config.StringParser)
	AppMasterKey = appMasterKey
	Otel         = config.LoadEnv(otel, false, config.BoolParser)

	GrpcPort     = config.LoadEnv(grpcPort, grpcPortDefault, config.IntParser)
	GrpcUrl      = grpcUrl
	GrpcTestPort = config.LoadEnv(grpcTestPort, grpcPortDefault, config.IntParser)
	GrpcTestUrl  = grpcTestUrl
	GrpcPing     = config.LoadEnv(grpcPing, grpcDefaultPing, config.DurationParser)

	GcloudProjectId = gcloudProjectId
)
