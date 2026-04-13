#!/bin/bash
# Sets up environment variables for local development and testing. Each variable uses the
# assign-if-unset pattern (${VAR:=default}), so pre-exported values are preserved and only
# missing ones are filled in. Source this file before running any local service or test command.

# ---- Ports ----
# Ports are allocated randomly via get-port-please to avoid conflicts when multiple instances
# run concurrently. Override by exporting a value before sourcing.

REST_PORT="${REST_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export REST_PORT
printf "Exposing REST on port %s\n" "${REST_PORT}"
GRPC_PORT="${GRPC_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export GRPC_PORT
printf "Exposing gRPC on port %s\n" "${GRPC_PORT}"
POSTGRES_PORT="${POSTGRES_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export POSTGRES_PORT

# ---- Credentials ----

# Dummy master key for local use only — never use this value in production or any shared environment.
export APP_MASTER_KEY="${APP_MASTER_KEY:="fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"}"

# ---- Service URLs ----

export GRPC_URL="${GRPC_URL:="localhost:${GRPC_PORT}"}"
export REST_URL="${REST_URL:="http://localhost:${REST_PORT}"}"

# ---- Database (Postgres) ----

export POSTGRES_USER="${POSTGRES_USER:="postgres"}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:="postgres"}"
export POSTGRES_DB="${POSTGRES_DB:="json-keys"}"
export POSTGRES_HOST="${POSTGRES_HOST:="localhost"}"
# Full DSN assembled from the individual fields above; override directly to use a non-standard connection string.
export POSTGRES_DSN="${POSTGRES_DSN:="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"}"
