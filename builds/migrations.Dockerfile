# Runs the migrations job: applies pending database schema migrations.
FROM docker.io/library/golang:1.26.4-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd/migrations ./cmd/migrations
COPY ./internal/config ./internal/config
COPY ./internal/models/migrations ./internal/models/migrations

RUN go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/

FROM docker.io/library/alpine:3.24.0

COPY --from=builder /migrations /migrations

CMD ["/migrations"]
