package main

import (
	"context"
	"fmt"
	"net"

	"github.com/samber/lo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	"github.com/a-novel/service-json-keys/v2/internal/handlers/protogen"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	"github.com/a-novel/service-json-keys/v2/internal/services"
)

// Runs the main gRPC server.
func main() {
	cfg := config.AppPresetDefault
	ctx := context.Background()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	ctx = lo.Must(lib.NewMasterKeyContext(ctx, cfg.App.MasterKey))
	ctx = lo.Must(postgres.NewContext(ctx, config.PostgresPresetDefault))

	repositoryJwkSearch := dao.NewJwkSearch()
	repositoryJwkSelect := dao.NewJwkSelect()

	serviceJwkExtract := services.NewJwkExtract()
	serviceJwkSearch := services.NewJwkSearch(repositoryJwkSearch, serviceJwkExtract)
	serviceJwkSelect := services.NewJwkSelect(repositoryJwkSelect, serviceJwkExtract)
	serviceExportLocal := services.NewJwkExportLocal(serviceJwkSearch)
	serviceJwkSource := lo.Must(services.NewJwkPrivateSource(serviceExportLocal, config.JwkPresetDefault))
	serviceJwkProducer := lo.Must(services.NewJwkProducers(serviceJwkSource, config.JwkPresetDefault))
	serviceClaimsSign := services.NewClaimsSign(serviceJwkProducer, config.JwkPresetDefault)

	handlerStatus := handlers.NewStatus()
	handlerClaimsSign := handlers.NewClaimsSign(serviceClaimsSign)
	handlersJwkGet := handlers.NewJwkGet(serviceJwkSelect)
	handlersJwkList := handlers.NewJwkList(serviceJwkSearch)

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
			cfg.Logger.UnaryInterceptor(),
			cfg.Logger.PanicUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpcf.BaseContextStreamInterceptor(ctxInterceptor),
			cfg.Logger.StreamInterceptor(),
			cfg.Logger.PanicStreamInterceptor(),
		),
	)

	grpcf.SetEchoServers(server, cfg.Grpc.Ping)

	protogen.RegisterStatusServiceServer(server, handlerStatus)
	protogen.RegisterClaimsSignServiceServer(server, handlerClaimsSign)
	protogen.RegisterJwkGetServiceServer(server, handlersJwkGet)
	protogen.RegisterJwkListServiceServer(server, handlersJwkList)

	reflection.Register(server)

	defer server.GracefulStop()
	defer server.Stop()

	err := server.Serve(listener)
	if err != nil {
		panic(err)
	}
}
