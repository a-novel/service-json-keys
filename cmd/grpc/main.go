// Command grpc runs the private gRPC server of the JSON Keys service, which other services call
// to sign tokens and read keys. It admits only the callers its service tokens allow, and refuses to
// start without a token issuer unless SERVICE_ISSUER_OPTIONAL opts a local stack out. Signing
// decrypts private keys, so the server refuses to start without APP_MASTER_KEY.
//
// cmd/rest serves the public keys over REST.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
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

	masterKey := lo.Must(lib.ParseMasterKey(cfg.App.MasterKey))

	// A server started during a planned downtime refuses work, so it leaves the database alone.
	if !downtime.Started(cfg.App.DowntimeStart, time.Now()) {
		ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))
		defer cfg.Postgres.Close()
	}

	server := lo.Must(newServer(ctx, cfg, masterKey))

	log.Println("Starting gRPC server on :" + strconv.Itoa(cfg.Grpc.Port))

	lo.Must0(grpcf.ListenAndServe(ctx, cfg.Grpc, server))
}

// newServer builds the gRPC server. Each call carries the database of ctx, and passes the
// service-token gate first, but for the health service. The gate refuses an empty issuer unless cfg
// opts a local stack out, so the server never starts open by mistake.
func newServer(ctx context.Context, cfg config.App, masterKey *[32]byte) (*grpc.Server, error) {
	serviceToken, err := grpcf.ServiceTokenGate(cfg.ServiceToken, httpf.NewPoolClient(httpf.PoolOptions{}))
	if err != nil {
		return nil, fmt.Errorf("service token gate: %w", err)
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoJwkSearch := dao.NewPgJwkSearch()
	daoJwkSelect := dao.NewPgJwkSelect()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	serviceJwkExtract := core.NewJwkExtract(masterKey)
	serviceJwkSearch := core.NewJwkSearch(daoJwkSearch, serviceJwkExtract)
	serviceJwkSelect := core.NewJwkSelect(daoJwkSelect, serviceJwkExtract)

	// The signing chain: a cached private-key source feeds the per-usage producer plugins, so
	// ClaimsSign signs tokens without hitting the database on every request.
	serviceJwkProducer := lo.Must(core.NewJwkProducers(serviceJwkSearch, jwkconfig.PresetDefault))
	serviceClaimsSign := core.NewClaimsSign(serviceJwkProducer, jwkconfig.PresetDefault)

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

	server := grpcf.NewGRPCServer(ctx, cfg.GrpcLogger, cfg.App.DowntimeStart, nil, serviceToken...)

	grpcf.RegisterHealth(ctx, server)

	jsonkeysv2.RegisterStatusServiceServer(server, handlerStatus)
	jsonkeysv2.RegisterClaimsSignServiceServer(server, handlerClaimsSign)
	jsonkeysv2.RegisterJwkGetServiceServer(server, handlerJwkGet)
	jsonkeysv2.RegisterJwkListServiceServer(server, handlerJwkList)

	reflection.Register(server)

	return server, nil
}
