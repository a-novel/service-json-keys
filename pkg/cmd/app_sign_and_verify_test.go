package cmdpkg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/models/api"
	"github.com/a-novel/service-json-keys/pkg"
)

type testClaims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func testAppSignAndVerify(_ context.Context, t *testing.T, client *apimodels.Client) {
	t.Helper()

	signer := pkg.NewClaimsSigner(client)
	verifier, err := pkg.NewClaimsVerifier[testClaims](client, models.DefaultJWKSConfig)
	require.NoError(t, err)

	claims := testClaims{
		Username: "testuser",
		Email:    "test@email.com",
	}

	signedToken, err := signer.SignClaims(t.Context(), models.KeyUsageAuth, claims)
	require.NoError(t, err)
	require.NotEmpty(t, signedToken)

	verifiedClaims, err := verifier.VerifyClaims(t.Context(), models.KeyUsageAuth, signedToken, nil)
	require.NoError(t, err)

	require.Equal(t, &claims, verifiedClaims)
}
