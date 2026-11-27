package core_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/v2/jwa"
	"github.com/a-novel-kit/jwt/v2/jwk"

	"github.com/a-novel/service-json-keys/v2/internal/core"
)

func TestNewAPIKeySelect(t *testing.T) {
	t.Parallel()

	const (
		purpose      = "test-purpose"
		otherPurpose = "other-purpose"
	)

	// RFC 8037 appendix A.3 gives this public key and its thumbprint.
	const rfcKeyID = "kPrK_qmxVWaYVA9wwBF6Iuo3vVzz7TxHCTwXBygrS4k"

	rfcKey, err := base64.RawURLEncoding.DecodeString("11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo")
	require.NoError(t, err)

	activeKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	retiredKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	type expectKey struct {
		id          string
		purpose     string
		publicKey   ed25519.PublicKey
		deactivated bool
	}

	testCases := []struct {
		name string

		settings map[string]string

		expect    []expectKey
		expectErr error
	}{
		{
			name: "Success/NoSetting",
		},
		{
			name:     "Success/EmptySetting",
			settings: map[string]string{purpose: " \n"},
		},
		{
			name:     "Success/Thumbprint",
			settings: map[string]string{purpose: mustPublicKeyPEM(t, ed25519.PublicKey(rfcKey))},

			expect: []expectKey{{id: rfcKeyID, purpose: purpose, publicKey: rfcKey}},
		},
		{
			name: "Success/ActiveFirst",
			settings: map[string]string{
				purpose: mustPublicKeyPEM(t, activeKey) + "\n" + mustPublicKeyPEM(t, retiredKey) + "\n",
			},

			expect: []expectKey{
				{id: thumbprintOf(activeKey), purpose: purpose, publicKey: activeKey},
				{id: thumbprintOf(retiredKey), purpose: purpose, publicKey: retiredKey, deactivated: true},
			},
		},
		{
			name: "Success/TwoPurposes",
			settings: map[string]string{
				purpose:      mustPublicKeyPEM(t, activeKey),
				otherPurpose: mustPublicKeyPEM(t, retiredKey),
			},

			expect: []expectKey{
				{id: thumbprintOf(activeKey), purpose: purpose, publicKey: activeKey},
				{id: thumbprintOf(retiredKey), purpose: otherPurpose, publicKey: retiredKey},
			},
		},
		{
			name:     "Error/NotPEM",
			settings: map[string]string{purpose: "not a key"},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			name:     "Error/TrailingData",
			settings: map[string]string{purpose: mustPublicKeyPEM(t, activeKey) + "trailing"},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			name:     "Error/TextBeforeKey",
			settings: map[string]string{purpose: "# rotated on 2026-11-27\n" + mustPublicKeyPEM(t, activeKey)},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			// The new key lost its last line, as a truncated paste does. The key meant to retire must
			// not take its place.
			name: "Error/MalformedActiveKey",
			settings: map[string]string{
				purpose: strings.TrimSuffix(mustPublicKeyPEM(t, activeKey), "-----END PUBLIC KEY-----\n") +
					"\n" + mustPublicKeyPEM(t, retiredKey),
			},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			name:     "Error/PrivateKey",
			settings: map[string]string{purpose: mustPrivateKeyPEM(t, privateKey)},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			name: "Error/NotAPublicKey",
			settings: map[string]string{
				purpose: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("not a key")})),
			},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			name:     "Error/NotEd25519",
			settings: map[string]string{purpose: mustPublicKeyPEM(t, &ecdsaKey.PublicKey)},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			name:     "Error/ListedTwice",
			settings: map[string]string{purpose: mustPublicKeyPEM(t, activeKey) + mustPublicKeyPEM(t, activeKey)},

			expectErr: core.ErrInvalidAPIKeys,
		},
		{
			name: "Error/TwoPurposes",
			settings: map[string]string{
				purpose:      mustPublicKeyPEM(t, activeKey),
				otherPurpose: mustPublicKeyPEM(t, activeKey),
			},

			expectErr: core.ErrInvalidAPIKeys,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service, err := core.NewAPIKeySelect(testCase.settings)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				// A setting may hold a private key by mistake, so the error quotes none of its lines.
				for _, setting := range testCase.settings {
					for line := range strings.Lines(setting) {
						if line = strings.TrimSpace(line); line != "" {
							require.NotContains(t, err.Error(), line)
						}
					}
				}

				return
			}

			for _, expect := range testCase.expect {
				key, err := service.Exec(t.Context(), &core.APIKeySelectRequest{ID: expect.id})
				require.NoError(t, err)

				require.Equal(t, expect.id, key.ID)
				require.Equal(t, expect.purpose, key.Purpose)
				require.Equal(t, expect.deactivated, key.Deactivated)
				require.False(t, key.Leaked)

				require.Equal(t, expect.id, key.Key.KID)
				require.Equal(t, jwa.KeyOps{jwa.KeyOpVerify}, key.Key.KeyOps)

				// The served key is one the jwt library reads as this public key, with no private half.
				private, public, err := jwk.ConsumeED25519(key.Key)
				require.NoError(t, err)
				require.Nil(t, private)
				require.Equal(t, expect.publicKey, public.Key())
			}

			_, err = service.Exec(t.Context(), &core.APIKeySelectRequest{ID: "unknown"})
			require.ErrorIs(t, err, core.ErrAPIKeyNotFound)
		})
	}
}

func TestAPIKeySelect(t *testing.T) {
	t.Parallel()

	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	service, err := core.NewAPIKeySelect(map[string]string{"test-purpose": mustPublicKeyPEM(t, publicKey)})
	require.NoError(t, err)

	testCases := []struct {
		name string

		request *core.APIKeySelectRequest

		expectErr error
	}{
		{
			name:    "Success",
			request: &core.APIKeySelectRequest{ID: thumbprintOf(publicKey)},
		},
		{
			name:    "Error/NotFound",
			request: &core.APIKeySelectRequest{ID: thumbprintOf(publicKey) + "A"},

			expectErr: core.ErrAPIKeyNotFound,
		},
		{
			name:    "Error/NoID",
			request: &core.APIKeySelectRequest{},

			expectErr: core.ErrAPIKeyNotFound,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			key, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr == nil {
				require.Equal(t, testCase.request.ID, key.ID)
			}
		})
	}
}

// thumbprintOf returns the JWK thumbprint of an Ed25519 public key, built here from RFC 7638 §3 and
// RFC 8037 §2 so the test does not check the service against the function it calls.
func thumbprintOf(publicKey ed25519.PublicKey) string {
	digest := sha256.Sum256([]byte(
		`{"crv":"Ed25519","kty":"OKP","x":"` + base64.RawURLEncoding.EncodeToString(publicKey) + `"}`,
	))

	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// mustPublicKeyPEM returns a public key in PEM, as a setting of system API keys holds it.
func mustPublicKeyPEM(t *testing.T, publicKey any) string {
	t.Helper()

	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

// mustPrivateKeyPEM returns a private key in PEM, which no setting may hold.
func mustPrivateKeyPEM(t *testing.T, privateKey any) string {
	t.Helper()

	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		panic(err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}
