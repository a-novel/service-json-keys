#!/bin/bash

# Runs the Go package integration tests against a live gRPC service.
# Starts the containerized gRPC service via podman-compose, then runs all /pkg packages.

set -e

APP_NAME="service-json-keys-integration-test"
PODMAN_FILE="$PWD/builds/podman-compose.integration-test.grpc.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
cleanup() {
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap cleanup INT EXIT

. "$PWD/scripts/setup-env.sh"

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

# Wait for the gRPC service to pass its healthcheck before running tests.
# The standalone image has a grpcurl HEALTHCHECK that covers both the service and any startup migrations.
elapsed=0
until [ "$(podman inspect --format '{{.State.Health.Status}}' "${APP_NAME}_service-json-keys_1" 2>/dev/null)" = "healthy" ]; do
    elapsed=$((elapsed + 1))
    if [ "$elapsed" -ge 60 ]; then
        printf "error: gRPC service did not become healthy within 60s\n" >&2
        exit 1
    fi
    sleep 1
done

# shellcheck disable=SC2046
PACKAGES="$(go list ./... | grep /pkg)"
go tool -modfile=gotestsum.mod gotestsum --format pkgname -- -count=1 -cover $PACKAGES

# Normal execution: containers are shut down by the EXIT trap.
