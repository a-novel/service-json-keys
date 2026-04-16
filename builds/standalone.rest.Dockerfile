# Standalone REST server image for local development. Applies migrations and rotates keys
# automatically before starting the server, so the database does not need to be pre-initialized.
# For production deployments, use the base REST image (rest.Dockerfile) instead.
FROM docker.io/library/golang:1.26.2-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd/rest ./cmd/rest
COPY ./cmd/migrations ./cmd/migrations
COPY ./cmd/rotate-keys ./cmd/rotate-keys
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

RUN go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/ && \
    go build -ldflags="-s -w" -trimpath -o /rotate-keys ./cmd/rotate-keys/

FROM docker.io/library/alpine:3.23.4

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations
COPY --from=builder /rotate-keys /rotate-keys

# Alpine ships BusyBox wget — no extra package needed for the healthcheck.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD wget -qO /dev/null http://localhost:8080/ping || exit 1

ENV REST_PORT=8080

# REST port.
EXPOSE 8080

# Apply migrations and rotate keys, then start the server.
CMD ["sh", "-c", "/migrations && /rotate-keys && /rest"]
