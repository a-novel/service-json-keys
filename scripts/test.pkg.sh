#!/bin/bash

APP_NAME="service-json-keys-integration-test"
PODMAN_FILE="$PWD/builds/podman-compose.integration-test.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap int_handler INT

# Setup test containers.
podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

# shellcheck disable=SC2046
PACKAGES="$(go list ./... | grep /pkg)"
go tool gotestsum --format pkgname -- -count=1 -cover $PACKAGES

# Normal execution: containers are shut down.
podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
