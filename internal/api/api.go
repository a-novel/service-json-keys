package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-json-keys/models/api"
)

var ErrInternalServerError = &apimodels.UnexpectedErrorStatusCode{
	StatusCode: http.StatusInternalServerError,
	Response:   apimodels.UnexpectedError{Error: "internal server error"},
}

type API struct {
	apimodels.UnimplementedHandler

	SelectKeyService  SelectKeyService
	SearchKeysService SearchKeysService
	SignClaimsService SignClaimsService
}

func (api *API) NewError(ctx context.Context, err error) *apimodels.UnexpectedErrorStatusCode {
	// no-op
	if err == nil {
		return nil
	}

	logger := otel.Logger()
	logger.ErrorContext(ctx, fmt.Sprintf("security error: %v", err))

	return ErrInternalServerError
}
