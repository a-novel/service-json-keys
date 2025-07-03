# This image is a convenience for tests and development. It is not suitable for production use.
FROM golang:alpine AS builder

WORKDIR /app

COPY ../cmd ./cmd
COPY ../config ./config
COPY ../internal/api ./internal/api
COPY ../internal/adapters ./internal/adapters
COPY ../internal/dao ./internal/dao
COPY ../internal/lib ./internal/lib
COPY ../internal/services ./internal/services
COPY ../migrations ./migrations
COPY ../models ./models
COPY ../go.mod ./go.mod
COPY ../go.sum ./go.sum

RUN go mod download

RUN go build -o /api cmd/api/main.go
RUN go build -o /rotatekeys cmd/rotatekeys/main.go

FROM alpine:latest

WORKDIR /

COPY --from=builder /api /api
COPY --from=builder /rotatekeys /rotatekeys

RUN apk --update add curl

ENV HOST="0.0.0.0"

ENV PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=1s --timeout=5s --retries=20 --start-period=1s \
  CMD curl -f http://localhost:8080/v1/ping || exit 1

# Run
CMD ["sh", "-c", "/rotatekeys && /api"]
