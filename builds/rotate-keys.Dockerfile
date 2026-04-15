# Runs the key rotation job: generates new JSON Web Keys for each usage where the rotation
# interval has elapsed. Requires a database with migrations already applied; does not
# require a running server.
FROM docker.io/library/golang:1.26.2-alpine AS builder

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

FROM docker.io/library/alpine:3.23.4

WORKDIR /

COPY --from=builder /rotate-keys /rotate-keys

ARG DEBIAN_FRONTEND=noninteractive

# Rotate the JSON Web Keys for all configured usages.
CMD ["/rotate-keys"]
