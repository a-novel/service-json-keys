FROM docker.io/library/golang:1.25.2-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rotatekeys" "./cmd/rotatekeys"
COPY ./internal/adapters ./internal/adapters
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/api ./internal/api
COPY ./internal/services ./internal/services
COPY ./models ./models
COPY "./pkg/cmd" "./pkg/cmd"

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /job cmd/rotatekeys/main.go

FROM docker.io/library/alpine:3.22.2

WORKDIR /

COPY --from=builder /job /job

# Make sure the migrations are run before the job starts.
CMD ["/job"]
