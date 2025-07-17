# This image is a convenience for tests and development. It is not suitable for production use.
FROM docker.io/library/golang:alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/api" "./cmd/api"
COPY "./cmd/rotatekeys" "./cmd/rotatekeys"
COPY "./cmd/migrations" "./cmd/migrations"
COPY ./internal/api ./internal/api
COPY ./internal/adapters ./internal/adapters
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./migrations ./migrations
COPY ./models ./models
COPY "./pkg/cmd" "./pkg/cmd"

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /api cmd/api/main.go
RUN go build -o /rotatekeys cmd/rotatekeys/main.go
RUN go build -o /migrations cmd/migrations/main.go

FROM docker.io/library/alpine:latest

WORKDIR /

COPY --from=builder /api /api
COPY --from=builder /rotatekeys /rotatekeys
COPY --from=builder /migrations /migrations

# ======================================================================================================================
# Healthcheck.
# ======================================================================================================================
RUN apk --update add curl

HEALTHCHECK --interval=1s --timeout=5s --retries=20 --start-period=1s \
  CMD curl -f http://localhost:8080/v1/ping || exit 1

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
ENV PORT=8080

EXPOSE 8080

ENV HOST="0.0.0.0"

# Make sure the migrations are run before the API starts.
CMD ["sh", "-c", "/migrations && /rotatekeys && /api"]
