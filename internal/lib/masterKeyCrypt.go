package lib

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

var (
	// ErrInvalidMasterKey is returned when the master key is missing or malformed.
	ErrInvalidMasterKey = errors.New("invalid master key")
	// ErrInvalidSecret is returned when decryption fails due to an invalid key or tampered ciphertext.
	ErrInvalidSecret = errors.New("invalid secret")
	// ErrInvalidCiphertext is returned when the ciphertext is too short to hold a valid secretbox message.
	ErrInvalidCiphertext = errors.New("ciphertext too short")
)

const (
	// MasterKeyLength is the expected length, in bytes, of the master encryption key.
	MasterKeyLength = 32
	// NonceLength is the length, in bytes, of the NaCl secretbox nonce prepended to each encrypted payload.
	NonceLength = 24
)

// ParseMasterKey decodes the hex-encoded master key.
//
// The master key is a secure, 32-byte random secret that encrypts private JSON Web Keys in the
// database. Rotating it permanently invalidates every private key encrypted with the old value.
func ParseMasterKey(masterKeyHex string) (*[MasterKeyLength]byte, error) {
	masterKey, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode master key: %w", err)
	}

	if len(masterKey) != MasterKeyLength {
		return nil, fmt.Errorf(
			"%w: expected %d bytes, got %d bytes",
			ErrInvalidMasterKey, MasterKeyLength, len(masterKey),
		)
	}

	// secretbox takes the key as a fixed-size array.
	return (*[MasterKeyLength]byte)(masterKey), nil
}

// EncryptMasterKey JSON-marshals data and encrypts it with masterKey. The returned ciphertext
// includes an embedded nonce and can only be decrypted by [DecryptMasterKey]. A nil masterKey
// returns [ErrInvalidMasterKey].
func EncryptMasterKey(masterKey *[MasterKeyLength]byte, data any) ([]byte, error) {
	if masterKey == nil {
		return nil, ErrInvalidMasterKey
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

	return secretbox.Seal(nonce[:], serializedData, &nonce, masterKey), nil
}

// DecryptMasterKey decrypts a ciphertext produced by [EncryptMasterKey] with masterKey, then
// JSON-unmarshals the result into output, which must be a non-nil pointer. A nil masterKey returns
// [ErrInvalidMasterKey].
func DecryptMasterKey(masterKey *[MasterKeyLength]byte, data []byte, output any) error {
	if masterKey == nil {
		return ErrInvalidMasterKey
	}

	// Secretbox requires a 24-byte nonce prefix plus at least the 16-byte Poly1305 tag.
	if len(data) < NonceLength+secretbox.Overhead {
		return fmt.Errorf("decrypt data: %w", ErrInvalidCiphertext)
	}

	decryptNonce := [NonceLength]byte(data[:NonceLength])

	decrypted, ok := secretbox.Open(nil, data[NonceLength:], &decryptNonce, masterKey)
	if !ok {
		return fmt.Errorf("decrypt data: %w", ErrInvalidSecret)
	}

	err := json.Unmarshal(decrypted, output)
	if err != nil {
		return fmt.Errorf("unmarshal data: %w", err)
	}

	return nil
}
