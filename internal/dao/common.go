package dao

import "errors"

// ErrJwkNotFound is returned when no active key matches the request. It is joined with the
// database error that caused it.
var ErrJwkNotFound = errors.New("jwk not found")
