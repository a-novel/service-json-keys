# Runs the key rotation job: generates new JSON Web Keys for each usage where the rotation
# interval has elapsed. Requires a database with migrations already applied; does not
# require a running server.
FROM docker.io/library/golang:1.26.4-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/rotate-keys ./cmd/rotate-keys
COPY ./internal/config ./internal/config
COPY ./internal/dao ./internal/dao
COPY ./internal/core ./internal/core
COPY ./internal/lib ./internal/lib

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rotate-keys ./cmd/rotate-keys/

FROM docker.io/library/alpine:3.24.1

COPY --from=builder /rotate-keys /rotate-keys

CMD ["/rotate-keys"]
