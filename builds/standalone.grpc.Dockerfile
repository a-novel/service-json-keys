# Standalone gRPC server image for local development. Applies migrations and rotates keys
# automatically before starting the server, so the database does not need to be pre-initialized.
# For production deployments, use the base gRPC image (grpc.Dockerfile) instead.
FROM docker.io/library/golang:1.26.2-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN GOBIN=/usr/local/bin go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.3

COPY ./cmd/grpc ./cmd/grpc
COPY ./cmd/migrations ./cmd/migrations
COPY ./cmd/rotate-keys ./cmd/rotate-keys
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

RUN go build -ldflags="-s -w" -trimpath -o /grpc ./cmd/grpc/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/ && \
    go build -ldflags="-s -w" -trimpath -o /rotate-keys ./cmd/rotate-keys/

FROM docker.io/library/alpine:3.23.4

COPY --from=builder /grpc /grpc
COPY --from=builder /migrations /migrations
COPY --from=builder /rotate-keys /rotate-keys
COPY --from=builder /usr/local/bin/grpcurl /usr/local/bin/grpcurl

HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD grpcurl --plaintext -d '' localhost:8080 grpc.health.v1.Health/Check || exit 1

ENV GRPC_PORT=8080

# gRPC port.
EXPOSE 8080
# TLS port.
EXPOSE 443

# Apply migrations and rotate keys, then start the server.
CMD ["sh", "-c", "/migrations && /rotate-keys && /grpc"]
