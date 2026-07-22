# Standalone REST server image for local development. Applies migrations and rotates keys
# automatically before starting the server, so the database needs no pre-initialization.
# Production deployments use the base REST image (rest.Dockerfile).
FROM docker.io/library/golang:1.26.5-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/rest ./cmd/rest
COPY ./cmd/migrations ./cmd/migrations
COPY ./cmd/rotate-keys ./cmd/rotate-keys
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/core ./internal/core
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/ && \
    go build -ldflags="-s -w" -trimpath -o /rotate-keys ./cmd/rotate-keys/

FROM docker.io/library/alpine:3.24.1

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations
COPY --from=builder /rotate-keys /rotate-keys

# Alpine ships BusyBox wget — no extra package needed for the healthcheck.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD wget -qO /dev/null http://localhost:8080/ping || exit 1

ENV REST_PORT=8080

EXPOSE 8080

CMD ["sh", "-c", "/migrations && /rotate-keys && /rest"]
