package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/uptrace/bun/driver/pgdriver"

	configparser "github.com/a-novel-kit/golib/config"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
)

const postgresDialTimeout = 3 * time.Minute

var errPostgresPasswordEmpty = errors.New("POSTGRES_PASSWORD is empty while POSTGRES_HOST is set")

// PostgresConnection describes how to reach PostgreSQL. Host selects the discrete
// fields; an empty Host selects the legacy DSN fallback.
type PostgresConnection struct {
	DSN        string
	Host       string
	Port       int
	User       string
	Password   string
	Database   string
	TLSEnabled bool
}

// LoadPostgres reads and validates the PostgreSQL connection and pool configuration.
func LoadPostgres() (*postgrespresets.Default, error) {
	var (
		port         = env.PostgresPortDefault
		tlsEnabled   = env.PostgresTLSEnabledDefault
		maxOpenConns = env.PostgresMaxOpenConnsDefault
		maxIdleConns = env.PostgresMaxIdleConnsDefault
		loadErrors   []error
	)

	loadValue(&loadErrors, &port, "POSTGRES_PORT", env.PostgresPortDefault, configparser.IntParser)
	loadValue(
		&loadErrors,
		&tlsEnabled,
		"POSTGRES_TLS_ENABLED",
		env.PostgresTLSEnabledDefault,
		configparser.BoolParser,
	)
	loadValue(
		&loadErrors,
		&maxOpenConns,
		"POSTGRES_MAX_OPEN_CONNS",
		env.PostgresMaxOpenConnsDefault,
		configparser.IntParser,
	)
	loadValue(
		&loadErrors,
		&maxIdleConns,
		"POSTGRES_MAX_IDLE_CONNS",
		env.PostgresMaxIdleConnsDefault,
		configparser.IntParser,
	)

	connection := PostgresConnection{
		DSN:        env.Get("POSTGRES_DSN"),
		Host:       env.Get("POSTGRES_HOST"),
		Port:       port,
		User:       env.Get("POSTGRES_USER"),
		Password:   env.Get("POSTGRES_PASSWORD"),
		Database:   env.Get("POSTGRES_DATABASE"),
		TLSEnabled: tlsEnabled,
	}
	if connection.Host != "" && connection.Password == "" {
		loadErrors = append(loadErrors, errPostgresPasswordEmpty)
	}

	err := errors.Join(loadErrors...)
	if err != nil {
		return nil, fmt.Errorf("load postgres config: %w", err)
	}

	return NewPostgresPreset(connection, maxOpenConns, maxIdleConns), nil
}

// NewPostgresPreset returns a PostgreSQL preset whose pool is bounded before it opens.
func NewPostgresPreset(
	connection PostgresConnection,
	maxOpenConns int,
	maxIdleConns int,
) *postgrespresets.Default {
	options := append(connection.options(), pgdriver.WithDialTimeout(postgresDialTimeout))
	preset := postgrespresets.NewDefault(options...)
	preset.MaxOpenConns = maxOpenConns
	preset.MaxIdleConns = maxIdleConns

	return preset
}

func (connection PostgresConnection) options() []pgdriver.Option {
	if connection.Host == "" {
		return []pgdriver.Option{pgdriver.WithDSN(connection.DSN)}
	}

	if connection.Password == "" {
		panic("postgres password is empty")
	}

	return []pgdriver.Option{
		pgdriver.WithAddr(net.JoinHostPort(connection.Host, strconv.Itoa(connection.Port))),
		pgdriver.WithUser(connection.User),
		pgdriver.WithPassword(connection.Password),
		pgdriver.WithDatabase(connection.Database),
		pgdriver.WithInsecure(!connection.TLSEnabled),
	}
}
