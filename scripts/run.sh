#!/bin/bash

# Runs the full application stack locally using podman-compose.
# Sources setup-env.sh for environment variables, then brings up all services.

set -e

APP_NAME="service-json-keys-local"
PODMAN_FILE="$PWD/builds/podman-compose.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
cleanup() {
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap cleanup INT EXIT

. "$PWD/scripts/setup-env.sh"

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up --build

# Normal execution: containers are shut down by the EXIT trap.
