package main

import (
	"context"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	sentryotel "github.com/getsentry/sentry-go/otel"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
)

const SentryFlushTimeout = 2 * time.Second

func main() {
	ctx := context.Background()

	err := sentry.Init(config.SentryClient)
	if err != nil {
		log.Fatalf("initialize sentry: %v", err)
	}
	defer sentry.Flush(SentryFlushTimeout)

	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(sentryotel.NewSentryPropagator())

	logger := sentry.NewLogger(ctx)
	logger.SetAttributes(
		attribute.String("app", "agora"),
		attribute.String("service", "json-keys"),
	)

	ctx, err = lib.NewAgoraContext(ctx, config.DSN)
	if err != nil {
		logger.Fatalf(ctx, "initialize agora context: %v", err)
	}

	span := sentry.StartSpan(ctx, "Job.RotateKeys")
	defer span.Finish()

	searchKeysDAO := dao.NewSearchKeysRepository()
	insertKeyDAO := dao.NewInsertKeyRepository()

	generateKeysService := services.NewGenerateKeyService(
		services.NewGenerateKeySource(searchKeysDAO, insertKeyDAO),
	)

	rotateKeyUsage := func(usage models.KeyUsage) {
		subSpan := sentry.StartSpan(span.Context(), "GenerateNewKey")
		defer subSpan.Finish()

		subSpan.SetData("usage", usage)

		keyID, err := generateKeysService.GenerateKey(subSpan.Context(), usage)
		if err != nil {
			subSpan.SetData("error", err.Error())
			logger.Errorf(subSpan.Context(), "generate key for usage %s: %v", usage, err)

			return
		}

		if keyID != nil {
			subSpan.SetData("keyID", keyID)
			logger.Infof(subSpan.Context(), "generated new key for usage %s: %s", usage, *keyID)

			return
		}

		subSpan.SetData("keyID", "nil")
	}

	for _, usage := range models.KnownKeyUsages {
		rotateKeyUsage(usage)
	}
}
