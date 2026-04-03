#!/bin/bash

REST_PORT="${REST_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export REST_PORT
printf "Exposing Rest on port %s\n" "${REST_PORT}"
GRPC_PORT="${GRPC_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export GRPC_PORT
printf "Exposing GRPC on port %s\n" "${GRPC_PORT}"
POSTGRES_PORT="${POSTGRES_PORT:="$(node -e 'console.log(await (await import("get-port-please")).getRandomPort())')"}"
export POSTGRES_PORT

# Dummy key, safe for exposure.
export APP_MASTER_KEY="${APP_MASTER_KEY:="fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca"}"

export GRPC_URL="${GRPC_URL:="localhost:${GRPC_PORT}"}"
export REST_URL="${REST_URL:="http://localhost:${REST_PORT}"}"
export POSTGRES_USER="${POSTGRES_USER:="postgres"}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:="postgres"}"
export POSTGRES_DB="${POSTGRES_DB:="json-keys"}"
export POSTGRES_HOST="${POSTGRES_HOST:="0.0.0.0"}"
export POSTGRES_DSN="${POSTGRES_DSN:="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"}"
