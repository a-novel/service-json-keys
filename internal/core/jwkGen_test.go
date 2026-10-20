package core_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"

	"github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/core"
	coremocks "github.com/a-novel/service-json-keys/v2/internal/core/mocks"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/lib"
	"github.com/a-novel/service-json-keys/v2/internal/lib/libtest"
)

// checkGeneratedPrivateKey checks that key is a base64 ciphertext of a JSON Web Key under the
// master key held by ctx.
func checkGeneratedPrivateKey(ctx context.Context, key string) error {
	decoded, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	var decrypted jwa.JWK

	err = lib.DecryptMasterKey(ctx, decoded, &decrypted)
	if err != nil {
		return fmt.Errorf("decrypt key: %w", err)
	}

	return nil
}

// checkGeneratedPublicKey checks that key is a base64 serialization of a JSON Web Key.
func checkGeneratedPublicKey(key string) error {
	decoded, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	var deserialized jwa.JWK

	err = deserialized.UnmarshalJSON(decoded)
	if err != nil {
		return fmt.Errorf("unmarshal key: %w", err)
	}

	return nil
}

// checkJwkInsertRequest reports whether request stores a well-formed generated key, failing the
// test on the first defect it finds.
func checkJwkInsertRequest(ctx context.Context, t *testing.T, request *dao.JwkInsertRequest) bool {
	t.Helper()

	if request == nil {
		t.Errorf("request is nil")

		return false
	}

	if request.ID == uuid.Nil {
		t.Error("expected KID to be set, got nil")

		return false
	}

	err := checkGeneratedPrivateKey(ctx, request.PrivateKey)
	if err != nil {
		t.Errorf("checking private key: %s", err)

		return false
	}

	if request.PublicKey != nil {
		err = checkGeneratedPublicKey(*request.PublicKey)
		if err != nil {
			t.Errorf("checking public key: %s", err)

			return false
		}
	}

	if request.Expiration.IsZero() {
		t.Error("expected expiration to be set, got zero")

		return false
	}

	if request.Expiration.Before(request.Now) {
		t.Error("expected expiration to be after creation date")

		return false
	}

	return true
}

// jwkGenExtracted is the key the extraction mock returns for a stored key of alg and kid.
func jwkGenExtracted(alg jwa.Alg, kid string) *core.Jwk {
	return &core.Jwk{
		JWKCommon: jwa.JWKCommon{
			KTY: "test-kty",
			Use: "test-use",
			Alg: alg,
			KID: kid,
		},
		Payload: json.RawMessage(`{"message":"hello world"}`),
	}
}

func TestJwkGen(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context(), libtest.MasterKey)
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type daoSearchMock struct {
		resp []*dao.Jwk
		err  error
	}

	type daoInsertMock struct {
		resp *dao.Jwk
		err  error
	}

	type serviceExtractMock struct {
		resp *core.Jwk
		err  error
	}

	type testCaseDef struct {
		name string

		request *core.JwkGenRequest

		daoSearchMock      *daoSearchMock
		daoInsertMock      *daoInsertMock
		serviceExtractMock *serviceExtractMock
		keys               map[string]*config.Jwk

		expect    *core.Jwk
		expectErr error
	}

	// The rotation interval is 12 hours: the stale key is past it, the recent key within it.
	rotation := config.JwkKey{TTL: 24 * time.Hour, Rotation: 12 * time.Hour, Cache: 6 * time.Hour}
	staleKey := &dao.Jwk{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		PrivateKey: "cHJpdmF0ZS1rZXktMQ",
		PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
		Usage:      "test-usage",
		CreatedAt:  time.Now().Add(-13 * time.Hour),
		ExpiresAt:  time.Now().Add(time.Hour),
	}
	recentKey := &dao.Jwk{
		ID:         staleKey.ID,
		PrivateKey: staleKey.PrivateKey,
		PublicKey:  staleKey.PublicKey,
		Usage:      staleKey.Usage,
		CreatedAt:  time.Now().Add(-11 * time.Hour),
		ExpiresAt:  staleKey.ExpiresAt,
	}
	insertedKey := &dao.Jwk{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		PrivateKey: "cHJpdmF0ZS1rZXktMQ",
		PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
		Usage:      "test-usage",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}

	testCases := []testCaseDef{
		{
			name: "Success/OldKeys",

			request: &core.JwkGenRequest{Usage: "test-usage"},
			keys:    testKeys(jwa.Ed25519, rotation),

			daoSearchMock:      &daoSearchMock{resp: []*dao.Jwk{staleKey}},
			daoInsertMock:      &daoInsertMock{resp: insertedKey},
			serviceExtractMock: &serviceExtractMock{resp: jwkGenExtracted(jwa.Ed25519, insertedKey.ID.String())},

			expect: jwkGenExtracted(jwa.Ed25519, insertedKey.ID.String()),
		},
		{
			name: "Success/RecentKeys",

			request: &core.JwkGenRequest{Usage: "test-usage"},
			keys:    testKeys(jwa.Ed25519, rotation),

			daoSearchMock:      &daoSearchMock{resp: []*dao.Jwk{recentKey}},
			serviceExtractMock: &serviceExtractMock{resp: jwkGenExtracted(jwa.Ed25519, recentKey.ID.String())},

			expect: jwkGenExtracted(jwa.Ed25519, recentKey.ID.String()),
		},
		{
			name: "Error/Extract",

			request: &core.JwkGenRequest{Usage: "test-usage"},
			keys:    testKeys(jwa.Ed25519, rotation),

			daoSearchMock:      &daoSearchMock{resp: []*dao.Jwk{staleKey}},
			daoInsertMock:      &daoInsertMock{resp: insertedKey},
			serviceExtractMock: &serviceExtractMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Insert",

			request: &core.JwkGenRequest{Usage: "test-usage"},
			keys:    testKeys(jwa.Ed25519, rotation),

			daoSearchMock: &daoSearchMock{resp: []*dao.Jwk{staleKey}},
			daoInsertMock: &daoInsertMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Search",

			request: &core.JwkGenRequest{Usage: "test-usage"},
			keys:    testKeys(jwa.Ed25519, rotation),

			daoSearchMock: &daoSearchMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/ConfigNotFound",

			request: &core.JwkGenRequest{Usage: "test-usage"},
			keys:    map[string]*config.Jwk{},

			daoSearchMock: &daoSearchMock{resp: []*dao.Jwk{}},

			expectErr: core.ErrConfigNotFound,
		},
	}

	for _, alg := range supportedAlgorithms {
		algCase := testCaseDef{
			name: "Success/" + string(alg),

			request: &core.JwkGenRequest{Usage: "test-usage"},
			keys:    testKeys(alg, rotation),

			daoSearchMock:      &daoSearchMock{},
			daoInsertMock:      &daoInsertMock{resp: insertedKey},
			serviceExtractMock: &serviceExtractMock{resp: jwkGenExtracted(alg, insertedKey.ID.String())},

			expect: jwkGenExtracted(alg, insertedKey.ID.String()),
		}

		testCases = append(testCases, algCase)
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSearch := coremocks.NewMockJwkGenDaoSearch(t)
			daoInsert := coremocks.NewMockJwkGenDaoInsert(t)
			serviceExtract := coremocks.NewMockJwkGenServiceExtract(t)

			if testCase.daoSearchMock != nil {
				daoSearch.EXPECT().
					Exec(mock.Anything, &dao.JwkSearchRequest{Usage: testCase.request.Usage}).
					Return(testCase.daoSearchMock.resp, testCase.daoSearchMock.err)
			}

			if testCase.daoInsertMock != nil {
				daoInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.JwkInsertRequest) bool {
						return checkJwkInsertRequest(ctx, t, request)
					})).
					Return(testCase.daoInsertMock.resp, testCase.daoInsertMock.err)
			}

			if testCase.serviceExtractMock != nil {
				var keyToExtract *dao.Jwk

				if testCase.daoInsertMock != nil {
					keyToExtract = testCase.daoInsertMock.resp
				} else if testCase.daoSearchMock != nil {
					keyToExtract = testCase.daoSearchMock.resp[0]
				} else {
					t.Fatal(
						"expected daoSearchMock or daoInsertMock to be defined for serviceExtractMock",
					)
				}

				serviceExtract.EXPECT().
					Exec(mock.Anything, &core.JwkExtractRequest{Jwk: keyToExtract, Private: true}).
					Return(testCase.serviceExtractMock.resp, testCase.serviceExtractMock.err)
			}

			service := core.NewJwkGen(
				daoSearch,
				daoInsert,
				serviceExtract,
				testCase.keys,
			)

			key, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, key)

			daoSearch.AssertExpectations(t)
			daoInsert.AssertExpectations(t)
			serviceExtract.AssertExpectations(t)
		})
	}
}
