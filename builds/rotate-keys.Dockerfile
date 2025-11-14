FROM docker.io/library/golang:1.25.4-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rotate-keys" "./cmd/rotate-keys"
COPY ./internal/config ./internal/config
COPY ./internal/dao ./internal/dao
COPY ./internal/services ./internal/services
COPY ./internal/lib ./internal/lib

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /rotate-keys cmd/rotate-keys/main.go

FROM docker.io/library/alpine:3.22.2

WORKDIR /

COPY --from=builder /rotate-keys /rotate-keys

# Make sure the migrations are run before the job starts.
CMD ["/rotate-keys"]
