// Command grpc runs the private gRPC server for the JSON-keys service.
//
// This server is the authenticated service-to-service API. It exposes token signing,
// key retrieval, and health check endpoints. Because signing requires access to private
// key material, the APP_MASTER_KEY environment variable must be set before starting the
// server.
//
// For the public read-only REST API, see cmd/rest.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/samber/lo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/config/env"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

	ctx = lo.Must(lib.NewMasterKeyContext(ctx, cfg.App.MasterKey))
	ctx = lo.Must(postgres.NewContext(ctx, config.PostgresPresetDefault))

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	repositoryJwkSearch := dao.NewPgJwkSearch()
	repositoryJwkSelect := dao.NewPgJwkSelect()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceJwkExtract := services.NewJwkExtract()
	serviceJwkSearch := services.NewJwkSearch(repositoryJwkSearch, serviceJwkExtract)
	serviceJwkSelect := services.NewJwkSelect(repositoryJwkSelect, serviceJwkExtract)

	// Build the signing chain: a cached private-key source feeds per-usage producer plugins,
	// which ClaimsSign uses to sign tokens without hitting the database on every request.
	serviceExportLocal := services.NewJwkExportLocal(serviceJwkSearch)
	serviceJwkSource := lo.Must(services.NewJwkPrivateSource(serviceExportLocal, config.JwkPresetDefault))
	serviceJwkProducer := lo.Must(services.NewJwkProducers(serviceJwkSource, config.JwkPresetDefault))
	serviceClaimsSign := services.NewClaimsSign(serviceJwkProducer, config.JwkPresetDefault)

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

	listenerConfig := new(net.ListenConfig)
	listener := lo.Must(listenerConfig.Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", cfg.Grpc.Port)))
	server := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		cfg.Otel.RpcInterceptor(),
		grpc.ChainUnaryInterceptor(
			grpcf.BaseContextUnaryInterceptor(ctxInterceptor),
			cfg.GrpcLogger.UnaryInterceptor(),
			cfg.GrpcLogger.PanicUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcf.BaseContextStreamInterceptor(ctxInterceptor),
			cfg.GrpcLogger.StreamInterceptor(),
			cfg.GrpcLogger.PanicStreamInterceptor(),
		),
	)

	grpcf.SetEchoServers(server, cfg.Grpc.Ping)

	protogen.RegisterStatusServiceServer(server, handlerStatus)
	protogen.RegisterClaimsSignServiceServer(server, handlerClaimsSign)
	protogen.RegisterJwkGetServiceServer(server, handlerJwkGet)
	protogen.RegisterJwkListServiceServer(server, handlerJwkList)

	reflection.Register(server)

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	log.Println("Starting gRPC server on :" + strconv.Itoa(cfg.Grpc.Port))

	go func() {
		err := server.Serve(listener)
		if err != nil {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gRPC server...")
	server.GracefulStop()
}
