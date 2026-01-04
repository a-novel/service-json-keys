package lib

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"

	"github.com/a-novel-kit/golib/otel"
)

var ErrInvalidSecret = errors.New("invalid secret")

const masterKeyNonceLength = 24

// EncryptMasterKey encrypts the input using the master key saved in the context.
func EncryptMasterKey(ctx context.Context, data any) ([]byte, error) {
	ctx, span := otel.Tracer().Start(ctx, "lib.EncryptMasterKey")
	defer span.End()

	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get master key: %w", err))
	}

	span.AddEvent("masterKey.retrieved")

	serializedData, err := json.Marshal(data)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("serialize data: %w", err))
	}

	span.AddEvent("data.serialized")

	// Generate a random nonce for encryption.
	var nonce [masterKeyNonceLength]byte

	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate nonce: %w", err))
	}

	span.AddEvent("nonce.generated")

	encrypted := secretbox.Seal(nonce[:], serializedData, &nonce, &secret)

	span.AddEvent("data.encrypted")

	return otel.ReportSuccess(span, encrypted), nil
}

// DecryptMasterKey decrypts the input using the master key saved in the context.
func DecryptMasterKey(ctx context.Context, data []byte, output any) error {
	ctx, span := otel.Tracer().Start(ctx, "lib.DecryptMasterKey")
	defer span.End()

	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get master key: %w", err))
	}

	span.AddEvent("masterKey.retrieved")

	// Validate minimum data length to prevent panic on slice access.
	if len(data) < masterKeyNonceLength {
		return otel.ReportError(span, fmt.Errorf(
			"%w: invalid data length %d, minimum %d bytes required",
			ErrInvalidSecret, len(data), masterKeyNonceLength,
		))
	}

	// Retrieve the nonce value from the source.
	var decryptNonce [masterKeyNonceLength]byte
	copy(decryptNonce[:], data[:masterKeyNonceLength])

	decrypted, ok := secretbox.Open(nil, data[masterKeyNonceLength:], &decryptNonce, &secret)
	if !ok {
		return otel.ReportError(span, fmt.Errorf("decrypt data: %w", ErrInvalidSecret))
	}

	span.AddEvent("data.decrypted")

	err = json.Unmarshal(decrypted, &output)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("unmarshal data: %w", err))
	}

	span.AddEvent("data.unmarshalled")
	otel.ReportSuccessNoContent(span)

	return nil
}
