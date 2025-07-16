package cmdpkg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/internal/adapters"
	"github.com/a-novel/service-json-keys/internal/api"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	"github.com/a-novel/service-json-keys/internal/services"
	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/api"
)

type AppAppConfig struct {
	Name      string `json:"name"      yaml:"name"`
	MasterKey string `json:"masterKey" yaml:"masterKey"`
}

type AppApiTimeoutsConfig struct {
	Read       time.Duration `json:"read"       yaml:"read"`
	ReadHeader time.Duration `json:"readHeader" yaml:"readHeader"`
	Write      time.Duration `json:"write"      yaml:"write"`
	Idle       time.Duration `json:"idle"       yaml:"idle"`
	Request    time.Duration `json:"request"    yaml:"request"`
}

type AppCorsConfig struct {
	AllowedOrigins   []string `json:"allowedOrigins"   yaml:"allowedOrigins"`
	AllowedHeaders   []string `json:"allowedHeaders"   yaml:"allowedHeaders"`
	AllowCredentials bool     `json:"allowCredentials" yaml:"allowCredentials"`
	MaxAge           int      `json:"maxAge"           yaml:"maxAge"`
}

type AppAPIConfig struct {
	Port           int                  `json:"port"           yaml:"port"`
	Timeouts       AppApiTimeoutsConfig `json:"timeouts"       yaml:"timeouts"`
	MaxRequestSize int64                `json:"maxRequestSize" yaml:"maxRequestSize"`
	Cors           AppCorsConfig        `json:"cors"           yaml:"cors"`
}

type AppConfig[Otel otel.Config, Pg postgres.Config] struct {
	App  AppAppConfig                              `json:"app"  yaml:"app"`
	API  AppAPIConfig                              `json:"api"  yaml:"api"`
	JWKS map[models.KeyUsage]*models.JSONKeyConfig `json:"jwks" yaml:"jwks"`

	Otel     Otel `json:"otel"     yaml:"otel"`
	Postgres Pg   `json:"postgres" yaml:"postgres"`
}

func App[Otel otel.Config, Pg postgres.Config](ctx context.Context, config AppConfig[Otel, Pg]) error {
	// =================================================================================================================
	// DEPENDENCIES
	// =================================================================================================================
	otel.SetAppName(config.App.Name)

	err := otel.InitOtel(config.Otel)
	if err != nil {
		return fmt.Errorf("init otel: %w", err)
	}
	defer config.Otel.Flush()

	ctx, err = lib.NewMasterKeyContext(ctx, config.App.MasterKey)
	if err != nil {
		return fmt.Errorf("new master key context: %w", err)
	}

	// Don't override the context if it already has a bun.IDB
	_, err = postgres.GetContext(ctx)
	if err != nil {
		ctx, err = postgres.NewContext(ctx, config.Postgres)
		if err != nil {
			return fmt.Errorf("init postgres: %w", err)
		}
	}

	// =================================================================================================================
	// DAO
	// =================================================================================================================

	searchKeysDAO := dao.NewSearchKeysRepository()
	selectKeyDAO := dao.NewSelectKeyRepository()

	// =================================================================================================================
	// SERVICES
	// =================================================================================================================

	searchKeysService := services.NewSearchKeysService(searchKeysDAO)
	selectKeyService := services.NewSelectKeyService(selectKeyDAO)

	privateSources, err := services.NewPrivateKeySources(
		adapters.NewPrivateKeySourcesLocalAdapter(searchKeysService),
		config.JWKS,
	)
	if err != nil {
		return fmt.Errorf("initialize private key sources: %w", err)
	}

	producers, err := services.NewProducers(privateSources, config.JWKS)
	if err != nil {
		return fmt.Errorf("initialize producers: %w", err)
	}

	signClaimsService := services.NewSignClaimsService(producers, config.JWKS)

	// =================================================================================================================
	// SETUP ROUTER
	// =================================================================================================================

	router := chi.NewRouter()

	router.Use(middleware.Recoverer)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(config.API.Timeouts.Request))
	router.Use(middleware.RequestSize(config.API.MaxRequestSize))
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
	router.Use(config.Otel.HTTPHandler())

	handler := &api.API{
		SelectKeyService:  selectKeyService,
		SearchKeysService: searchKeysService,
		SignClaimsService: signClaimsService,
	}

	apiServer, err := apimodels.NewServer(handler)
	if err != nil {
		return fmt.Errorf("new api server: %w", err)
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

	err = httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
