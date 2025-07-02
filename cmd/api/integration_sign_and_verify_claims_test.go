package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/config"
	"github.com/a-novel/service-json-keys/models"
	"github.com/a-novel/service-json-keys/pkg"
)

type testClaims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func TestSignAndVerifyClaims(t *testing.T) {
	client, err := pkg.NewAPIClient(t.Context(), fmt.Sprintf("http://127.0.0.1:%v/v1", config.API.Port))
	require.NoError(t, err)

	signer := pkg.NewClaimsSigner(client)
	verifier, err := pkg.NewClaimsVerifier[testClaims](client)
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
