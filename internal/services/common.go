package services

import "errors"

// ErrConfigNotFound is returned when no JWK configuration is registered for a given key usage.
var ErrConfigNotFound = errors.New("no config found for the requested usage")

// ErrJwkNotFound is returned when no active key matches the requested ID.
var ErrJwkNotFound = errors.New("jwk not found")
