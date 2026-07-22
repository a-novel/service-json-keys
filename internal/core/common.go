package core

import "errors"

// ErrConfigNotFound is returned when no JWK configuration is registered for a given key usage.
var ErrConfigNotFound = errors.New("no config found for the requested usage")

// ErrJwkNotFound is returned when no active key matches the requested ID.
var ErrJwkNotFound = errors.New("jwk not found")

// ErrReservedClaim is returned when the caller's claims name a registered JWT
// parameter. Those belong to the envelope the service stamps from the usage
// config, so a caller setting one is asking for a token that contradicts the
// terms it was signed under. It reports a malformed request, not a fault.
var ErrReservedClaim = errors.New("claims may not set a registered JWT parameter")
