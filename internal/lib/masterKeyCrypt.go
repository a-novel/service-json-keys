package lib

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
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
	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get master key: %w", err)
	}

	serializedData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("serialize data: %w", err)
	}

	var nonce [NonceLength]byte

	_, err = io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	encrypted := secretbox.Seal(nonce[:], serializedData, &nonce, &secret)

	return encrypted, nil
}

// DecryptMasterKey decrypts a ciphertext produced by [EncryptMasterKey] using the master key stored
// in the context, then JSON-unmarshals the result into output, which must be a non-nil pointer.
func DecryptMasterKey(ctx context.Context, data []byte, output any) error {
	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return fmt.Errorf("get master key: %w", err)
	}

	// Secretbox requires a 24-byte nonce prefix plus at least the 16-byte Poly1305 tag.
	if len(data) < NonceLength+secretbox.Overhead {
		return fmt.Errorf("decrypt data: %w", ErrInvalidCiphertext)
	}

	var decryptNonce [NonceLength]byte
	copy(decryptNonce[:], data[:NonceLength])

	decrypted, ok := secretbox.Open(nil, data[NonceLength:], &decryptNonce, &secret)
	if !ok {
		return fmt.Errorf("decrypt data: %w", ErrInvalidSecret)
	}

	err = json.Unmarshal(decrypted, output)
	if err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	return nil
}
