// Command grpc runs the private gRPC server for the JSON-keys service: the authenticated
// service-to-service API covering token signing and key retrieval. Signing needs the private
// key material, so APP_MASTER_KEY must be set before the server starts.
//
// For the public read-only REST API, see cmd/rest.
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
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

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

	processCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx := processCtx

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
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
		rpCtx = postgres.TransferContext(ctx, rpCtx)
		rpCtx = lib.TransferMasterKeyContext(ctx, rpCtx)

		return rpCtx
	}

	server := grpcf.NewServer(ctx, cfg.Otel, cfg.GrpcLogger, cfg.App.DowntimeStart, ctxInterceptor)

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

	err := grpcf.ListenAndServe(ctx, cfg.Grpc, server)

	stop()

	if err != nil {
		panic(err)
	}
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Println("Close Postgres: " + err.Error())
	}
}
