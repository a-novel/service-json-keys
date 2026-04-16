# Runs the JSON-keys gRPC server. Requires a database with migrations already applied.
FROM docker.io/library/golang:1.26.2-alpine AS builder

# CGO_ENABLED=0 produces a fully static binary with no C library dependency,
# which is required for safe execution on Alpine (musl libc).
ENV CGO_ENABLED=0

WORKDIR /app

# Download dependencies before copying source files so the module cache layer is
# reused on rebuilds where only source files change.
COPY go.mod go.sum ./
RUN go mod download

# Install grpcurl for the healthcheck. Placed before source COPY so the cached
# layer (download + compile) survives source-only rebuilds.
RUN GOBIN=/usr/local/bin go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.3

COPY ./cmd/grpc ./cmd/grpc
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

# -ldflags="-s -w" strips the symbol table and DWARF debug info, shrinking the binary.
# -trimpath removes local filesystem paths for reproducible builds.
RUN go build -ldflags="-s -w" -trimpath -o /grpc ./cmd/grpc/

FROM docker.io/library/alpine:3.23.4

COPY --from=builder /grpc /grpc
COPY --from=builder /usr/local/bin/grpcurl /usr/local/bin/grpcurl

HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD grpcurl --plaintext -d '' localhost:8080 grpc.health.v1.Health/Check || exit 1

ENV GRPC_PORT=8080

# gRPC port.
EXPOSE 8080
# TLS port.
EXPOSE 443

CMD ["/grpc"]
