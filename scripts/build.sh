#!/bin/bash

# Builds all Docker images for the service under the :local tag.
# Run this before using podman-compose locally so all images are available without a registry pull.

set -e

# ---- Database ----

podman build --format docker \
  -f ./builds/database.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/database:local .

# ---- Jobs ----

podman build --format docker \
  -f ./builds/migrations.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/jobs/migrations:local .
podman build --format docker \
  -f ./builds/rotate-keys.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/jobs/rotatekeys:local .

# ---- gRPC server ----

podman build --format docker \
  -f ./builds/grpc.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/grpc:local .
podman build --format docker \
  -f ./builds/standalone.grpc.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/standalone-grpc:local .

# ---- REST server ----

podman build --format docker \
  -f ./builds/rest.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/rest:local .
podman build --format docker \
  -f ./builds/standalone.rest.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/standalone-rest:local .
