FROM docker.io/library/golang:alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rotatekeys" "./cmd/rotatekeys"
COPY "./cmd/migrations" "./cmd/migrations"
COPY ./internal/adapters ./internal/adapters
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/api ./internal/api
COPY ./internal/services ./internal/services
COPY ./migrations ./migrations
COPY ./models ./models
COPY "./pkg/cmd" "./pkg/cmd"

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /job cmd/rotatekeys/main.go
RUN go build -o /migrations cmd/migrations/main.go

FROM docker.io/library/alpine:latest

WORKDIR /

COPY --from=builder /job /job
COPY --from=builder /migrations /migrations

# Make sure the migrations are run before the job starts.
CMD ["sh", "-c", "/migrations && /job"]
