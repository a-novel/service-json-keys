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

var (
	ErrInvalidSecret     = errors.New("invalid secret")
	ErrInvalidCiphertext = errors.New("ciphertext too short")
)

const NonceLength = 24

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
	var nonce [NonceLength]byte

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

	// Secretbox requires a 24-byte nonce prefix plus at least the 16-byte Poly1305 tag.
	if len(data) < NonceLength+secretbox.Overhead {
		return otel.ReportError(span, fmt.Errorf("decrypt data: %w", ErrInvalidCiphertext))
	}

	// Retrieve the nonce value from the source.
	var decryptNonce [NonceLength]byte
	copy(decryptNonce[:], data[:NonceLength])

	decrypted, ok := secretbox.Open(nil, data[NonceLength:], &decryptNonce, &secret)
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
