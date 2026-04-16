# Runs the JSON-keys REST server. Requires a database with migrations already applied.
FROM docker.io/library/golang:1.26.2-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd/rest ./cmd/rest
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

RUN go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/

FROM docker.io/library/alpine:3.23.4

COPY --from=builder /rest /rest

# Alpine ships BusyBox wget — no extra package needed for the healthcheck.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD wget -qO /dev/null http://localhost:8080/ping || exit 1

ENV REST_PORT=8080

# REST API port.
EXPOSE 8080

CMD ["/rest"]
