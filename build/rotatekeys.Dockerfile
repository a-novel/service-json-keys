FROM golang:alpine AS builder

WORKDIR /app

COPY "../cmd/rotatekeys" "./cmd/rotatekeys"
COPY ../internal/adapters ./internal/adapters
COPY ../internal/dao ./internal/dao
COPY ../internal/lib ./internal/lib
COPY ../internal/api ./internal/api
COPY ../internal/services ./internal/services
COPY ../migrations ./migrations
COPY ../models ./models
COPY "../pkg/cmd" "./pkg/cmd"
COPY ../go.mod ./go.mod
COPY ../go.sum ./go.sum

RUN go mod download

RUN go build -o /job cmd/rotatekeys/main.go

FROM gcr.io/distroless/base:latest

WORKDIR /

COPY --from=builder /job /job

# Run
CMD ["/job"]
