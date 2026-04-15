# Standalone REST server image for local development. Applies migrations and rotates keys
# automatically before starting the server, so the database does not need to be pre-initialized.
# For production deployments, use the base REST image (rest.Dockerfile) instead.
FROM docker.io/library/golang:1.26.2-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rest" "./cmd/rest"
COPY "./cmd/migrations" "./cmd/migrations"
COPY "./cmd/rotate-keys" "./cmd/rotate-keys"
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /rest cmd/rest/main.go
RUN go build -o /migrations cmd/migrations/main.go
RUN go build -o /rotate-keys cmd/rotate-keys/main.go

FROM docker.io/library/alpine:3.23.4

WORKDIR /

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations
COPY --from=builder /rotate-keys /rotate-keys

# ======================================================================================================================
# Healthcheck.
# ======================================================================================================================
RUN apk --update add curl

HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD curl -f http://localhost:8080/ping || exit 1

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
# Make sure the executable uses the default port.
ENV REST_PORT=8080

# REST port.
EXPOSE 8080

# Apply migrations and rotate keys, then start the server.
CMD ["sh", "-c", "/migrations && /rotate-keys && /rest"]
