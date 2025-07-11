package lib_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/internal/lib"
)

func TestMasterKeyCrypt(t *testing.T) { //nolint:tparallel
	masterKey := hex.EncodeToString([]byte("secret-master-key"))
	fakeMasterKey := hex.EncodeToString([]byte("fake-master-key"))

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
}
