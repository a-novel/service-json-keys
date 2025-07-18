package services_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/lib"
	"github.com/a-novel/service-json-keys/internal/services"
	servicesmocks "github.com/a-novel/service-json-keys/internal/services/mocks"
	testutils "github.com/a-novel/service-json-keys/internal/test"
	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/config"
)

func checkGeneratedPrivateKey(ctx context.Context, t *testing.T, key string) (*jwa.JWK, error) {
	t.Helper()

	// Decode base64 value.
	decoded, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// Decrypt.
	var decrypted jwa.JWK

	err = lib.DecryptMasterKey(ctx, decoded, &decrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt key: %w", err)
	}

	return &decrypted, nil
}

func checkGeneratedPublicKey(t *testing.T, key string) (*jwa.JWK, error) {
	t.Helper()

	// Decode base64 value.
	decoded, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// Unmarshal.
	var deserialized jwa.JWK

	err = deserialized.UnmarshalJSON(decoded)
	if err != nil {
		return nil, fmt.Errorf("unmarshal key: %w", err)
	}

	return &deserialized, nil
}

func TestGenerateKeys(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), testutils.TestMasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type searchKeyData struct {
		resp []*dao.KeyEntity
		err  error
	}

	type insertKeyData struct {
		resp *dao.KeyEntity
		err  error
	}

	testCases := []struct {
		name string

		usage models.KeyUsage

		searchKeyData *searchKeyData
		insertKeyData *insertKeyData

		expectKID *uuid.UUID
		expectErr error
	}{
		{
			name: "KeyUsageAuth",

			usage: models.KeyUsageAuth,

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageAuth,
						CreatedAt:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageAuth,
						CreatedAt:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			insertKeyData: &insertKeyData{
				resp: &dao.KeyEntity{},
			},
		},
		{
			name: "KeyUsageRefresh",

			usage: models.KeyUsageRefresh,

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageRefresh,
						CreatedAt:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageRefresh,
						CreatedAt:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			insertKeyData: &insertKeyData{
				resp: &dao.KeyEntity{},
			},
		},

		{
			name: "SearchErr",

			usage: models.KeyUsageAuth,

			searchKeyData: &searchKeyData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "InsertError",

			usage: models.KeyUsageAuth,

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageAuth,
						CreatedAt:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageAuth,
						CreatedAt:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			insertKeyData: &insertKeyData{
				err: errFoo,
			},

			expectErr: errFoo,
		},

		{
			name: "NoKeysYet",

			usage: models.KeyUsageAuth,

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{},
			},

			insertKeyData: &insertKeyData{
				resp: &dao.KeyEntity{},
			},
		},
		{
			name: "LastKeyRecent",

			usage: models.KeyUsageAuth,

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageAuth,
						CreatedAt:  time.Now().Add(-time.Minute),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
					{
						ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						PrivateKey: "cHJpdmF0ZS1rZXktMQ",
						PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
						Usage:      models.KeyUsageAuth,
						CreatedAt:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						ExpiresAt:  time.Now().Add(time.Hour),
					},
				},
			},

			expectKID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockGenerateKeySource(t)

			if testCase.searchKeyData != nil {
				source.EXPECT().
					SearchKeys(mock.Anything, testCase.usage).
					Return(testCase.searchKeyData.resp, testCase.searchKeyData.err)
			}

			expectKID := testCase.expectKID

			checkInsertData := func(data dao.InsertKeyData) bool {
				// Check for the KID to be present, and assign it to the expectKID variable.
				if data.ID == uuid.Nil {
					t.Error("expected KID to be set, got nil")

					return false
				}

				// Only expected a Key ID if no error is returned.
				if testCase.insertKeyData.err == nil {
					expectKID = &data.ID
				}

				// Ensure private key is encrypted.
				_, err = checkGeneratedPrivateKey(ctx, t, data.PrivateKey)
				if err != nil {
					t.Errorf("checking private key: %s", err)

					return false
				}

				_, err = checkGeneratedPublicKey(t, *data.PublicKey)
				if err != nil {
					t.Errorf("checking public key: %s", err)

					return false
				}

				if data.Expiration.IsZero() {
					t.Error("expected expiration to be set, got zero")

					return false
				}

				if data.Expiration.Before(data.Now) {
					t.Error("expected expiration to be after creation date")

					return false
				}

				return true
			}

			if testCase.insertKeyData != nil {
				source.EXPECT().
					InsertKey(mock.Anything, mock.MatchedBy(checkInsertData)).
					Return(testCase.insertKeyData.resp, testCase.insertKeyData.err)
			}

			service := services.NewGenerateKeyService(source, config.JWKSPresetDefault)

			kid, err := service.GenerateKey(ctx, testCase.usage)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, expectKID, kid)

			source.AssertExpectations(t)
		})
	}
}
