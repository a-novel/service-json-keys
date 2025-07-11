// Code generated by ogen, DO NOT EDIT.

package codegen

import (
	"context"

	ht "github.com/ogen-go/ogen/http"
)

// UnimplementedHandler is no-op Handler which returns http.ErrNotImplemented.
type UnimplementedHandler struct{}

var _ Handler = UnimplementedHandler{}

// GetPublicKey implements getPublicKey operation.
//
// Get a public key from its usage.
//
// GET /public-keys
func (UnimplementedHandler) GetPublicKey(ctx context.Context, params GetPublicKeyParams) (r GetPublicKeyRes, _ error) {
	return r, ht.ErrNotImplemented
}

// Healthcheck implements healthcheck operation.
//
// Returns a detailed report of the health of the service, including every dependency.
//
// GET /healthcheck
func (UnimplementedHandler) Healthcheck(ctx context.Context) (r HealthcheckRes, _ error) {
	return r, ht.ErrNotImplemented
}

// ListPublicKeys implements listPublicKeys operation.
//
// Get all public keys from the service that match a given usage.
//
// GET /public-keys/list
func (UnimplementedHandler) ListPublicKeys(ctx context.Context, params ListPublicKeysParams) (r ListPublicKeysRes, _ error) {
	return r, ht.ErrNotImplemented
}

// Ping implements ping operation.
//
// Check the status of the service. If the service is running, a successful response is returned.
//
// GET /ping
func (UnimplementedHandler) Ping(ctx context.Context) (r PingRes, _ error) {
	return r, ht.ErrNotImplemented
}

// SignClaims implements signClaims operation.
//
// Sign a payload using the configuration for the target usage.
//
// POST /payload/sign
func (UnimplementedHandler) SignClaims(ctx context.Context, req SignClaimsReq, params SignClaimsParams) (r SignClaimsRes, _ error) {
	return r, ht.ErrNotImplemented
}

// NewError creates *UnexpectedErrorStatusCode from error returned by handler.
//
// Used for common default response.
func (UnimplementedHandler) NewError(ctx context.Context, err error) (r *UnexpectedErrorStatusCode) {
	r = new(UnexpectedErrorStatusCode)
	return r
}
