package lib_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/lib"
)

func TestNewMasterKeyContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		envValue string

		expect    [32]byte
		expectErr error
	}{
		{
			name:     "ValidKey",
			envValue: "1f0f29d72e880eec55360ea14bc18dfcbcc1a771dcd45fa03f0e5c181fdb368c",
			expect: [32]byte{
				31, 15, 41, 215, 46, 136, 14, 236, 85, 54, 14, 161, 75, 193, 141, 252, 188, 193, 167, 113, 220, 212,
				95, 160, 63, 14, 92, 24, 31, 219, 54, 140,
			},
		},
		{
			name: "LongerKey",
			envValue: "1f0f29d72e880eec55360ea14bc18dfcbcc1a771dcd45fa03f0e5c181fdb368c664fb4329736f23566d4d5a2ba8af" +
				"98375a88a0907ba34bf715901942df2b580",
			expectErr: lib.ErrInvalidMasterKey,
		},
		{
			name:      "ShorterKey",
			envValue:  "087a92fbcde7afd24bab23ba428df42e1eb8d6197b677509",
			expectErr: lib.ErrInvalidMasterKey,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, err := lib.NewMasterKeyContext(t.Context(), testCase.envValue)

			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)

				return
			}

			require.NoError(t, err)

			value, err := lib.MasterKeyContext(ctx)
			require.NoError(t, err)
			require.Equal(t, testCase.expect, value)

			nCtx := context.TODO()
			nCtx = lib.TransferMasterKeyContext(ctx, nCtx)

			transferredValue, err := lib.MasterKeyContext(nCtx)
			require.NoError(t, err)
			require.Equal(t, testCase.expect, transferredValue)
		})
	}
}

func TestMasterKeyContextMissing(t *testing.T) {
	t.Parallel()

	// Context without a master key should return ErrInvalidMasterKey.
	ctx := context.Background()

	_, err := lib.MasterKeyContext(ctx)
	require.ErrorIs(t, err, lib.ErrInvalidMasterKey)
}
