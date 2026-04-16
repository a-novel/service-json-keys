#!/bin/bash

# Runs the TypeScript/JS integration tests against a live REST service.
# Starts the containerized REST service, waits for it to be ready, then runs pnpm test.

set -e

APP_NAME="service-json-keys-integration-test-rest"
PODMAN_FILE="$PWD/builds/podman-compose.integration-test.rest.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
cleanup() {
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap cleanup INT EXIT

. "$PWD/scripts/setup-env.sh"

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

# Poll the /ping endpoint until the REST service is ready or the timeout is reached.
# curl -sf: -s suppresses progress, -f returns non-zero on HTTP errors (4xx/5xx).
elapsed=0
until curl -sf "${REST_URL}/ping" > /dev/null 2>&1; do
    elapsed=$((elapsed + 1))
    if [ "$elapsed" -ge 60 ]; then
        printf "error: REST service did not become ready within 60s\n" >&2
        exit 1
    fi
    printf "Waiting for REST service on port %s... (%ds)\n" "${REST_PORT}" "$elapsed"
    sleep 1
done

pnpm test || podman logs "${APP_NAME}_service-json-keys_1"

# Normal execution: containers are shut down by the EXIT trap.
