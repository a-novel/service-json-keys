package main

import (
	"context"
	"log"
	"os"

	"github.com/a-novel/golib/postgres"
	postgrespresets "github.com/a-novel/golib/postgres/presets"

	"github.com/a-novel/service-json-keys/migrations"
)

func main() {
	_, err := postgres.InitPostgres(context.Background(), postgrespresets.DefaultConfig{
		DSN:        os.Getenv("POSTGRES_DSN"),
		Migrations: migrations.Migrations,
	})
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
}
