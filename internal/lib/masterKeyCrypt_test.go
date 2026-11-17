package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

func TestParseMasterKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		masterKeyHex string

		expect    *[lib.MasterKeyLength]byte
		expectErr error
	}{
		{
			name:         "Success",
			masterKeyHex: "1f0f29d72e880eec55360ea14bc18dfcbcc1a771dcd45fa03f0e5c181fdb368c",
			expect: &[lib.MasterKeyLength]byte{
				31, 15, 41, 215, 46, 136, 14, 236, 85, 54, 14, 161, 75, 193, 141, 252, 188, 193, 167, 113, 220, 212,
				95, 160, 63, 14, 92, 24, 31, 219, 54, 140,
			},
		},
		{
			name: "Error/TooLong",
			masterKeyHex: "1f0f29d72e880eec55360ea14bc18dfcbcc1a771dcd45fa03f0e5c181fdb368c664fb4329736f23566d4d5a2ba8af" +
				"98375a88a0907ba34bf715901942df2b580",
			expectErr: lib.ErrInvalidMasterKey,
		},
		{
			name:         "Error/TooShort",
			masterKeyHex: "087a92fbcde7afd24bab23ba428df42e1eb8d6197b677509",
			expectErr:    lib.ErrInvalidMasterKey,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			masterKey, err := lib.ParseMasterKey(testCase.masterKeyHex)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, masterKey)
		})
	}
}

func TestMasterKeyCrypt(t *testing.T) {
	t.Parallel()

	// 32-byte keys (64 hex chars).
	masterKey, err := lib.ParseMasterKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	require.NoError(t, err)

	fakeMasterKey, err := lib.ParseMasterKey("fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")
	require.NoError(t, err)

	data := map[string]any{"foo": "bar"}

	encrypted, err := lib.EncryptMasterKey(masterKey, data)
	require.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		var decrypted map[string]any

		require.NoError(t, lib.DecryptMasterKey(masterKey, encrypted, &decrypted))
		require.Equal(t, data, decrypted)
	})

	t.Run("Error/WrongKey", func(t *testing.T) {
		t.Parallel()

		var decrypted map[string]any

		require.ErrorIs(t, lib.DecryptMasterKey(fakeMasterKey, encrypted, &decrypted), lib.ErrInvalidSecret)
		require.Nil(t, decrypted)
	})

	t.Run("Error/MissingKey", func(t *testing.T) {
		t.Parallel()

		var decrypted map[string]any

		_, err := lib.EncryptMasterKey(nil, data)
		require.ErrorIs(t, err, lib.ErrInvalidMasterKey)
		require.ErrorIs(t, lib.DecryptMasterKey(nil, encrypted, &decrypted), lib.ErrInvalidMasterKey)
		require.Nil(t, decrypted)
	})

	t.Run("Error/TooShort", func(t *testing.T) {
		t.Parallel()

		var decrypted map[string]any

		// Ciphertext must be at least NonceLength (24) + secretbox.Overhead (16) = 40 bytes.
		shortData := make([]byte, 10)

		require.ErrorIs(t, lib.DecryptMasterKey(masterKey, shortData, &decrypted), lib.ErrInvalidCiphertext)
		require.Nil(t, decrypted)
	})
}
