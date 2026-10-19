// Command rest runs the public REST server for the JSON-keys service. It serves read-only
// JSON Web Key endpoints, unauthenticated, so any client can fetch public keys for local
// token verification. Public-key retrieval requires database access but no master key.
//
// For the private authenticated gRPC API (including token signing), see cmd/grpc.
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

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/config/env"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/handlers"
)

func main() {
	cfg := config.AppPresetDefault

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	otel.SetAppName(cfg.App.Name)

	lo.Must0(otel.Init(cfg.Otel))
	defer cfg.Otel.Flush()

	if env.GcloudProjectId == "" {
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}

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

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerPing := handlers.NewRestPing()
	handlerHealth := handlers.NewRestHealth()
	handlerJwkList := handlers.NewRestJwkList(serviceJwkSearch, cfg.Logger)
	handlerJwkGet := handlers.NewRestJwkGet(serviceJwkSelect, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewRouter(cfg.Rest, cfg.Otel, cfg.RestLogger, cfg.App.DowntimeStart)

	router.Route("/v2", func(api chi.Router) {
		api.Get("/ping", handlerPing.ServeHTTP)
		api.Get("/healthcheck", handlerHealth.ServeHTTP)
		api.Get("/jwks", handlerJwkList.ServeHTTP)
		api.Get("/jwk", handlerJwkGet.ServeHTTP)
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	log.Println("Starting REST server on :" + strconv.Itoa(cfg.Rest.Port))

	lo.Must0(httpf.ListenAndServe(ctx, cfg.Rest, router))
}

func closeDatabase(database io.Closer) {
	err := database.Close()
	if err != nil {
		log.Println("Close Postgres: " + err.Error())
	}
}
