// Command grpc runs the private gRPC server of the JSON Keys service, which other services call
// to sign tokens and read keys. Once service tokens are configured, it admits only the callers they
// allow. Otherwise it authenticates no caller, so only a private network may reach it. Signing
// decrypts private keys, so the server refuses to start without APP_MASTER_KEY.
//
// cmd/rest serves the public keys over REST.
package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/samber/lo"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/sentryf"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/config/env"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

func main() {
	cfg := config.AppPresetDefault

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	lo.Must0(sentryf.Init(cfg.Sentry))

	defer sentryf.Flush(config.SentryFlushTimeout)

	if env.OTLPEndpoint == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	ctx = lo.Must(lib.NewMasterKeyContext(ctx, cfg.App.MasterKey))

	// A server started during a planned downtime refuses work, so it leaves the database alone.
	if !downtime.Started(cfg.App.DowntimeStart, time.Now()) {
		ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))

		database := lo.Must(cfg.Postgres.DB(ctx))
		defer closeDatabase(database)
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoJwkSearch := dao.NewPgJwkSearch()
	daoJwkSelect := dao.NewPgJwkSelect()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceJwkExtract := core.NewJwkExtract()
	serviceJwkSearch := core.NewJwkSearch(daoJwkSearch, serviceJwkExtract)
	serviceJwkSelect := core.NewJwkSelect(daoJwkSelect, serviceJwkExtract)

	// The signing chain: a cached private-key source feeds the per-usage producer plugins, so
	// ClaimsSign signs tokens without hitting the database on every request.
	serviceExportLocal := core.NewJwkExportLocal(serviceJwkSearch)
	serviceJwkProducer := lo.Must(core.NewJwkProducers(serviceExportLocal, config.JwkPresetDefault))
	serviceClaimsSign := core.NewClaimsSign(serviceJwkProducer, config.JwkPresetDefault)

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerStatus := handlers.NewGrpcStatus()
	handlerClaimsSign := handlers.NewGrpcClaimsSign(serviceClaimsSign)
	handlerJwkGet := handlers.NewGrpcJwkGet(serviceJwkSelect)
	handlerJwkList := handlers.NewGrpcJwkList(serviceJwkSearch)

	// =================================================================================================================
	// SERVER
	// =================================================================================================================

	ctxInterceptor := func(rpCtx context.Context) context.Context {
		return lib.TransferMasterKeyContext(ctx, postgres.TransferContext(ctx, rpCtx))
	}

	serviceToken := handlers.GrpcServiceToken(cfg.ServiceToken, httpf.NewPoolClient(httpf.PoolOptions{}))

	server := grpcf.NewServer(ctx, cfg.Otel, cfg.GrpcLogger, cfg.App.DowntimeStart, ctxInterceptor, serviceToken...)

	healthcheck := grpcf.RegisterEchoServers(server)
	healthcheck.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	go func() {
		<-ctx.Done()
		healthcheck.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}()

	jsonkeysv2.RegisterStatusServiceServer(server, handlerStatus)
	jsonkeysv2.RegisterClaimsSignServiceServer(server, handlerClaimsSign)
	jsonkeysv2.RegisterJwkGetServiceServer(server, handlerJwkGet)
	jsonkeysv2.RegisterJwkListServiceServer(server, handlerJwkList)

	reflection.Register(server)

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	log.Println("Starting gRPC server on :" + strconv.Itoa(cfg.Grpc.Port))

	lo.Must0(grpcf.ListenAndServe(ctx, cfg.Grpc, server))
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Println("Close Postgres: " + err.Error())
	}
}
