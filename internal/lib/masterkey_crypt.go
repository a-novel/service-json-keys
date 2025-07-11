package lib

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/nacl/secretbox"

	"github.com/a-novel/golib/otel"
)

var (
	ErrInvalidSecret = errors.New("invalid secret")

	ErrEncryptMasterKey = errors.New("EncryptMasterKey")
	ErrDecryptMasterKey = errors.New("DecryptMasterKey")
)

func NewErrEncryptMasterKey(err error) error {
	return errors.Join(err, ErrEncryptMasterKey)
}

func NewErrDecryptMasterKey(err error) error {
	return errors.Join(err, ErrDecryptMasterKey)
}

// EncryptMasterKey encrypts the input using the master key.
func EncryptMasterKey(ctx context.Context, data any) ([]byte, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.EncryptMasterKey")
	defer span.End()

	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, NewErrEncryptMasterKey(fmt.Errorf("get master key: %w", err)))
	}

	span.AddEvent("masterKey.retrieved")

	serializedData, err := json.Marshal(data)
	if err != nil {
		return nil, otel.ReportError(span, NewErrEncryptMasterKey(fmt.Errorf("serialize data: %w", err)))
	}

	span.AddEvent("data.serialized")

	var nonce [24]byte

	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		return nil, otel.ReportError(span, NewErrEncryptMasterKey(fmt.Errorf("generate nonce: %w", err)))
	}

	span.AddEvent("nonce.generated")

	encrypted := secretbox.Seal(nonce[:], serializedData, &nonce, &secret)

	span.AddEvent("data.encrypted")

	return otel.ReportSuccess(span, encrypted), nil
}

// DecryptMasterKey decrypts the input using the master key.
func DecryptMasterKey(ctx context.Context, data []byte, output any) error {
	ctx, span := otel.Tracer().Start(ctx, "lib.DecryptMasterKey")
	defer span.End()

	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get master key: %w", err))
	}

	span.AddEvent("masterKey.retrieved")

	var decryptNonce [24]byte

	copy(decryptNonce[:], data[:24])

	decrypted, ok := secretbox.Open(nil, data[24:], &decryptNonce, &secret)
	if !ok {
		return otel.ReportError(span, NewErrDecryptMasterKey(fmt.Errorf("decrypt data: %w", ErrInvalidSecret)))
	}

	span.AddEvent("data.decrypted")

	err = json.Unmarshal(decrypted, &output)
	if err != nil {
		return otel.ReportError(span, NewErrDecryptMasterKey(fmt.Errorf("unmarshal data: %w", err)))
	}

	span.AddEvent("data.unmarshalled")
	span.SetStatus(codes.Ok, "")

	return nil
}
