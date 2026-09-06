package env

import (
	"fmt"
	"os"
	"time"
)

// Default values used when the corresponding environment variable is absent.
const (
	AppNameDefault = "service-json-keys"

	GrpcPortDefault            = 8080
	GrpcDefaultPing            = 5 * time.Second
	GrpcTimeoutShutdownDefault = 30 * time.Second

	RestPortDefault              = 8080
	RestTimeoutReadDefault       = 15 * time.Second
	RestTimeoutReadHeaderDefault = 3 * time.Second
	RestTimeoutWriteDefault      = 30 * time.Second
	RestTimeoutIdleDefault       = 60 * time.Second
	RestTimeoutRequestDefault    = 60 * time.Second
	RestTimeoutShutdownDefault   = 30 * time.Second
	RestMaxRequestSizeDefault    = 2 << 20 // 2 MiB
	CorsAllowCredentialsDefault  = false
	CorsMaxAgeDefault            = 3600

	// PostgresMaxOpenConnsDefault leaves capacity for jobs and operator sessions.
	PostgresMaxOpenConnsDefault = 20
	// PostgresMaxIdleConnsDefault keeps burst connections ready for reuse.
	PostgresMaxIdleConnsDefault = 20
	PostgresPortDefault         = 5432
	PostgresTLSEnabledDefault   = true
)

var (
	// CorsAllowedOriginsDefault permits every origin when no deployment policy is supplied.
	CorsAllowedOriginsDefault = []string{"*"}
	// CorsAllowedHeadersDefault permits every request header when no deployment policy is supplied.
	CorsAllowedHeadersDefault = []string{"*"}
)

// Get returns the raw value of a service-prefixed environment variable.
func Get(name string) string {
	return os.Getenv(variableName(name))
}

// Load parses a service-prefixed environment variable and returns fallback when it is unset.
func Load[T any](name string, fallback T, parser func(string) (T, error)) (T, error) {
	value := Get(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := parser(value)
	if err != nil {
		return fallback, fmt.Errorf("parse %s as %T: %w", variableName(name), fallback, err)
	}

	return parsed, nil
}

func variableName(name string) string {
	return os.Getenv("SERVICE_JSON_KEYS_ENV_PREFIX") + name
}
