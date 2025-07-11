package lib

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/a-novel/golib/otel"
)

type masterKeyContext struct{}

var ErrInvalidMasterKey = errors.New("invalid master key")

func NewMasterKeyContext(ctx context.Context, masterKeyRaw string) (context.Context, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.NewMasterKeyContext")
	defer span.End()

	masterKeyBytes, err := hex.DecodeString(masterKeyRaw)
	if err != nil {
		return ctx, otel.ReportError(span, fmt.Errorf("decode master key: %w", err))
	}

	var masterKey [32]byte

	copy(masterKey[:], masterKeyBytes)

	return otel.ReportSuccess(span, context.WithValue(ctx, masterKeyContext{}, masterKey)), nil
}

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
