# This image runs a job that rotates the JSON web keys for each usage in the application.
#
# It requires a patched database instance to run properly. It does not require a running
# server.
FROM docker.io/library/golang:1.26.0-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rotate-keys" "./cmd/rotate-keys"
COPY ./internal/config ./internal/config
COPY ./internal/dao ./internal/dao
COPY ./internal/services ./internal/services
COPY ./internal/lib ./internal/lib

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /rotate-keys cmd/rotate-keys/main.go

FROM docker.io/library/alpine:3.23.3

WORKDIR /

COPY --from=builder /rotate-keys /rotate-keys

# Rotate the JSON web keys for all usages.
CMD ["/rotate-keys"]
