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
	// ErrInvalidSecret is returned when decryption fails due to an invalid key or tampered ciphertext.
	ErrInvalidSecret = errors.New("invalid secret")
	// ErrInvalidCiphertext is returned when the ciphertext is too short to hold a valid secretbox message.
	ErrInvalidCiphertext = errors.New("ciphertext too short")
)

// NonceLength is the length, in bytes, of the NaCl secretbox nonce prepended to each encrypted payload.
const NonceLength = 24

// EncryptMasterKey JSON-marshals data and encrypts it using the master key stored in the context.
// The returned ciphertext includes an embedded nonce and can only be decrypted by [DecryptMasterKey].
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

// DecryptMasterKey decrypts a ciphertext produced by [EncryptMasterKey] using the master key stored
// in the context, then JSON-unmarshals the result into output, which must be a non-nil pointer.
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
