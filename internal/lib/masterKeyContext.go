package lib

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
)

// masterKeyContext is the context key used to store the master encryption key.
type masterKeyContext struct{}

// ErrInvalidMasterKey is returned when the master key is absent from the context or malformed.
var ErrInvalidMasterKey = errors.New("invalid master key")

// MasterKeyLength is the expected length, in bytes, of the master encryption key.
const MasterKeyLength = 32

// NewMasterKeyContext parses the provided master encryption key and makes it available in
// the context. The master key must be provided as a hex-encoded string.
//
// The master key is a secure, 32-byte random secret used to encrypt private JSON Web Keys
// in the database. It must be kept secret and generated with a cryptographically secure
// random source.
//
// Rotating this secret permanently invalidates all private keys encrypted with the old value,
// so update it with care.
func NewMasterKeyContext(ctx context.Context, masterKeyRaw string) (context.Context, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.NewMasterKeyContext")
	defer span.End()

	masterKeyBytes, err := hex.DecodeString(masterKeyRaw)
	if err != nil {
		return ctx, otel.ReportError(span, fmt.Errorf("decode master key: %w", err))
	}

	if len(masterKeyBytes) != MasterKeyLength {
		return ctx, otel.ReportError(span, fmt.Errorf(
			"%w: expected %d bytes, got %d bytes",
			ErrInvalidMasterKey, MasterKeyLength, len(masterKeyBytes),
		))
	}

	// Convert the raw master key to a fixed 32-byte array.
	// This is required for use with the secretbox package (see golang.org/x/crypto/nacl/secretbox).
	var masterKey [MasterKeyLength]byte
	copy(masterKey[:], masterKeyBytes)

	return otel.ReportSuccess(span, context.WithValue(ctx, masterKeyContext{}, masterKey)), nil
}

// MasterKeyContext returns the master key stored in the context.
// If no master key is present, [ErrInvalidMasterKey] is returned.
func MasterKeyContext(ctx context.Context) ([MasterKeyLength]byte, error) {
	masterKey, ok := ctx.Value(masterKeyContext{}).([MasterKeyLength]byte)

	if !ok {
		return [MasterKeyLength]byte{}, fmt.Errorf(
			"extract master key: %w: got type %T, expected %T",
			ErrInvalidMasterKey,
			ctx.Value(masterKeyContext{}), [MasterKeyLength]byte{},
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
	masterKey, ok := baseCtx.Value(masterKeyContext{}).([MasterKeyLength]byte)
	if !ok {
		return destCtx
	}

	return context.WithValue(destCtx, masterKeyContext{}, masterKey)
}
