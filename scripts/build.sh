#!/bin/bash

set -e

podman build --format docker \
  -f ./builds/database.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/database:local .

podman build --format docker \
  -f ./builds/migrations.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/jobs/migrations:local .
podman build --format docker \
  -f ./builds/rotate-keys.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/jobs/rotatekeys:local .

podman build --format docker \
  -f ./builds/grpc.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/grpc:local .
podman build --format docker \
  -f ./builds/standalone.Dockerfile \
  -t ghcr.io/a-novel/service-json-keys/standalone:local .
