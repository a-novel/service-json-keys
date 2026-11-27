// Command rest runs the public REST server of the JSON Keys service. It serves the public keys,
// API keys included, unauthenticated, so any client can verify tokens locally. It reads the database but never
// the master key.
//
// cmd/grpc serves the private API, token signing included.
package main

import (
	"context"
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

	// A server started during a planned downtime refuses work, so it leaves the database alone.
	if !downtime.Started(cfg.App.DowntimeStart, time.Now()) {
		ctx = lo.Must(postgres.NewContext(ctx, cfg.Postgres))
		defer cfg.Postgres.Close()
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	daoJwkSearch := dao.NewPgJwkSearch()
	daoJwkSelect := dao.NewPgJwkSelect()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	// The REST server holds no master key, so it reads public keys only.
	serviceJwkExtract := core.NewJwkExtract(nil)
	serviceJwkSearch := core.NewJwkSearch(daoJwkSearch, serviceJwkExtract)
	serviceJwkSelect := core.NewJwkSelect(daoJwkSelect, serviceJwkExtract)
	serviceAPIKeySelect := lo.Must(core.NewAPIKeySelect(cfg.APIKeys))

	// =================================================================================================================
	// HANDLERS
	// =================================================================================================================

	handlerHealth := httpf.Health(map[string]func(context.Context) error{
		"client:postgres": postgres.Health,
	})
	handlerJwkList := handlers.NewRestJwkList(serviceJwkSearch, cfg.Logger)
	handlerJwkGet := handlers.NewRestJwkGet(serviceJwkSelect, cfg.Logger)
	handlerAPIKeyGet := handlers.NewRestAPIKeyGet(serviceAPIKeySelect, cfg.Logger)

	// =================================================================================================================
	// ROUTER
	// =================================================================================================================

	router := httpf.NewMux(cfg.Rest, cfg.RestLogger, cfg.App.DowntimeStart)

	router.Route("/v2", func(api chi.Router) {
		api.Get("/ping", httpf.Ping)
		api.Get("/healthcheck", handlerHealth)
		api.Get("/jwks", handlerJwkList.ServeHTTP)
		api.Get("/jwk", handlerJwkGet.ServeHTTP)
		api.Get("/api-key", handlerAPIKeyGet.ServeHTTP)
	})

	// =================================================================================================================
	// RUN
	// =================================================================================================================

	log.Println("Starting REST server on :" + strconv.Itoa(cfg.Rest.Port))

	lo.Must0(httpf.ListenAndServe(ctx, cfg.Rest, router))
}
