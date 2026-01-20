package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

func TestMasterKeyCrypt(t *testing.T) { //nolint:tparallel
	// 32-byte keys (64 hex chars).
	masterKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	fakeMasterKey := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"

	ctxReal, err := lib.NewMasterKeyContext(t.Context(), masterKey)
	require.NoError(t, err)

	ctxFake, err := lib.NewMasterKeyContext(t.Context(), fakeMasterKey)
	require.NoError(t, err)

	data := map[string]any{"foo": "bar"}

	encrypted, err := lib.EncryptMasterKey(ctxReal, data)
	require.NoError(t, err)

	t.Run("DecryptOK", func(t *testing.T) {
		t.Parallel()

		var decrypted map[string]any

		require.NoError(t, lib.DecryptMasterKey(ctxReal, encrypted, &decrypted))
		require.Equal(t, data, decrypted)
	})

	t.Run("DecryptKO", func(t *testing.T) {
		t.Parallel()

		var decrypted map[string]any

		require.ErrorIs(t, lib.DecryptMasterKey(ctxFake, encrypted, &decrypted), lib.ErrInvalidSecret)
		require.Nil(t, decrypted)
	})

	t.Run("DecryptTooShort", func(t *testing.T) {
		t.Parallel()

		var decrypted map[string]any

		// Ciphertext must be at least NonceLength (24) + secretbox.Overhead (16) = 40 bytes.
		shortData := make([]byte, 10)

		require.ErrorIs(t, lib.DecryptMasterKey(ctxReal, shortData, &decrypted), lib.ErrInvalidCiphertext)
		require.Nil(t, decrypted)
	})
}
