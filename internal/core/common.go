package core

import (
	"errors"

	"github.com/a-novel/service-json-keys/v2/internal/config/jwkconfig"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
)

// ErrConfigNotFound is returned when no JWK configuration is registered for a given key usage.
var ErrConfigNotFound = jwkconfig.ErrConfigNotFound

// ErrJwkNotFound is returned when no active key matches the requested ID.
var ErrJwkNotFound = dao.ErrJwkNotFound

// ErrReservedClaim is returned when the caller's claims name a registered JWT parameter. The
// service stamps those from the usage config, so the request is malformed.
var ErrReservedClaim = errors.New("claims may not set a registered JWT parameter")
