package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"
	sentryhttp "github.com/getsentry/sentry-go/http"
	sentryotel "github.com/getsentry/sentry-go/otel"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/internal/adapters"
	"github.com/a-novel/service-json-keys/internal/api"
	"github.com/a-novel/service-json-keys/internal/api/codegen"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	"github.com/a-novel/service-json-keys/internal/services"
)

const (
	MaxRequestSize     = 2 << 20 // 2 MiB
	SentryFlushTimeout = 2 * time.Second
)

func main() {
	ctx := context.Background()
	// =================================================================================================================
	// LOAD DEPENDENCIES (EXTERNAL)
	// =================================================================================================================
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

	logger.Info(ctx, "starting application")

	ctx, err = lib.NewAgoraContext(ctx, config.DSN)
	if err != nil {
		logger.Fatalf(ctx, "initialize agora context: %v", err)
	}

	// =================================================================================================================
	// LOAD REPOSITORIES (INTERNAL)
	// =================================================================================================================

	// REPOSITORIES ----------------------------------------------------------------------------------------------------

	searchKeysDAO := dao.NewSearchKeysRepository()
	selectKeyDAO := dao.NewSelectKeyRepository()

	// SERVICES --------------------------------------------------------------------------------------------------------

	searchKeysService := services.NewSearchKeysService(searchKeysDAO)
	selectKeyService := services.NewSelectKeyService(selectKeyDAO)

	privateSources, err := services.NewPrivateKeySources(adapters.NewPrivateKeySourcesLocalAdapter(searchKeysService))
	if err != nil {
		logger.Fatalf(ctx, "initialize private key sources: %v", err)
	}

	producers, err := services.NewProducers(privateSources)
	if err != nil {
		logger.Fatalf(ctx, "initialize producers: %v", err)
	}

	signClaimsService := services.NewSignClaimsService(producers)

	// =================================================================================================================
	// SETUP ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	// MIDDLEWARES -----------------------------------------------------------------------------------------------------

	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(config.API.Timeouts.Request))
	router.Use(middleware.RequestSize(MaxRequestSize))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   config.API.Cors.AllowedOrigins,
		AllowedHeaders:   config.API.Cors.AllowedHeaders,
		AllowCredentials: config.API.Cors.AllowCredentials,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		MaxAge: config.API.Cors.MaxAge,
	}))

	sentryHandler := sentryhttp.New(sentryhttp.Options{})
	router.Use(sentryHandler.Handle)

	// RUN -------------------------------------------------------------------------------------------------------------

	handler := &api.API{
		SelectKeyService:  selectKeyService,
		SearchKeysService: searchKeysService,
		SignClaimsService: signClaimsService,
	}

	apiServer, err := codegen.NewServer(handler)
	if err != nil {
		logger.Fatalf(ctx, "start server: %v", err)
	}

	router.Mount("/v1/", http.StripPrefix("/v1", apiServer))

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(config.API.Port),
		Handler:           router,
		ReadTimeout:       config.API.Timeouts.Read,
		ReadHeaderTimeout: config.API.Timeouts.ReadHeader,
		WriteTimeout:      config.API.Timeouts.Write,
		IdleTimeout:       config.API.Timeouts.Idle,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	logger.SetAttributes(attribute.Int("server.port", config.API.Port))
	logger.Infof(ctx, "start http server on port %v", config.API.Port)

	err = httpServer.ListenAndServe()
	if err != nil {
		logger.Fatalf(ctx, "start http server: %v", err)
	}
}
