package lib

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/a-novel/golib/otel"
)

// Key used to identify the master key value in a context.
type masterKeyContext struct{}

var ErrInvalidMasterKey = errors.New("invalid master key")

// NewMasterKeyContext parses the provided master encryption key, and makes it available in
// the context. The master key must be provided as an hex encoded string.
//
// The master key is a secure, 32-byte random secret used to encrypt private JSON keys
// in the database. It must not leak and be random.
//
// Note that rotating this secret will doom all the private keys that have been encoded
// using the older version, so be cautious about updating its value.
func NewMasterKeyContext(ctx context.Context, masterKeyRaw string) (context.Context, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.NewMasterKeyContext")
	defer span.End()

	masterKeyBytes, err := hex.DecodeString(masterKeyRaw)
	if err != nil {
		return ctx, otel.ReportError(span, fmt.Errorf("decode master key: %w", err))
	}

	// Convert the raw master key to a fixed 32 bytes array.
	// This is required for usage with secretbox (see golang.org/x/crypto/nacl/secretbox).
	var masterKey [32]byte
	copy(masterKey[:], masterKeyBytes)

	return otel.ReportSuccess(span, context.WithValue(ctx, masterKeyContext{}, masterKey)), nil
}

// MasterKeyContext returns the master key saved in the current context. If the current context
// does not contain a master key, ErrInvalidMasterKey is thrown.
func MasterKeyContext(ctx context.Context) ([32]byte, error) {
	masterKey, ok := ctx.Value(masterKeyContext{}).([32]byte)

	if !ok {
		return [32]byte{}, fmt.Errorf(
			"extract master key: %w: got type %T, expected %T",
			ErrInvalidMasterKey,
			ctx.Value(masterKeyContext{}), [32]byte{},
		)
	}

	return masterKey, nil
}

// TransferMasterKeyContext passes the master key saved in the base context to a
// new context derived from the destination context. It returns the newly created
// context.
//
// If the base context does not contain any master key, this is a no-op, and the
// destination context is returned as-is.
func TransferMasterKeyContext(baseCtx, destCtx context.Context) context.Context {
	masterKey, ok := baseCtx.Value(masterKeyContext{}).([32]byte)
	if !ok {
		return destCtx
	}

	return context.WithValue(destCtx, masterKeyContext{}, masterKey)
}
