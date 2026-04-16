#!/bin/bash

# Runs the Go integration test suite in a containerized environment.
# Brings up the required services via podman-compose, applies migrations, then runs all internal packages.

set -e

APP_NAME="service-json-keys-test"
PODMAN_FILE="$PWD/builds/podman-compose.test.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
# Registered on EXIT (covers normal exit, set -e exits, and INT) — ERR is omitted to avoid
# double-firing with set -e.
cleanup() {
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap cleanup INT EXIT

. "$PWD/scripts/setup-env.sh"

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

# Wait for the database to accept connections before running migrations.
# The postgres container has a pg_isready HEALTHCHECK — poll its status rather than sleeping blindly.
elapsed=0
until [ "$(podman inspect --format '{{.State.Health.Status}}' "${APP_NAME}_postgres-json-keys_1" 2>/dev/null)" = "healthy" ]; do
    elapsed=$((elapsed + 1))
    if [ "$elapsed" -ge 30 ]; then
        printf "error: postgres container did not become healthy within 30s\n" >&2
        exit 1
    fi
    sleep 1
done

go run cmd/migrations/main.go

# shellcheck disable=SC2046
PACKAGES="$(go list ./... | grep internal | grep -v /mocks | grep -v /test | grep -v /protogen)"
go tool -modfile=gotestsum.mod gotestsum --format pkgname -- -count=1 -cover $PACKAGES

# Normal execution: containers are shut down by the EXIT trap.
