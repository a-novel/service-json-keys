package api

import (
	"context"
	"net/http"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-json-keys/internal/api/codegen"
)

var ErrInternalServerError = &codegen.UnexpectedErrorStatusCode{
	StatusCode: http.StatusInternalServerError,
	Response:   codegen.UnexpectedError{Error: "internal server error"},
}

type API struct {
	codegen.UnimplementedHandler

	SelectKeyService  SelectKeyService
	SearchKeysService SearchKeysService
	SignClaimsService SignClaimsService
}

func (api *API) NewError(ctx context.Context, err error) *codegen.UnexpectedErrorStatusCode {
	// no-op
	if err == nil {
		return nil
	}

	logger := sentry.NewLogger(ctx)
	logger.Errorf(ctx, "security error: %v", err)

	return ErrInternalServerError
}
