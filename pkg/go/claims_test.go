package servicejsonkeys_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/a-novel-kit/golib/grpcf"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
	"github.com/a-novel/service-json-keys/v2/pkg/go"
)

func TestClaimsVerifier(t *testing.T) {
	t.Parallel()

	client, err := servicejsonkeys.NewClient(env.GrpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer client.Close()

	type claims struct {
		Foo string `json:"foo"`
	}

	c := claims{
		Foo: "bar",
	}

	cp, err := grpcf.MarshalJSONAsAny(c)
	require.NoError(t, err)

	signRequest := &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuth,
		Payload: cp,
	}

	signed, err := client.ClaimsSign(t.Context(), signRequest)
	require.NoError(t, err)
	require.NotEmpty(t, signed)

	verifier, err := servicejsonkeys.NewClaimsVerifier[claims](client)
	require.NoError(t, err)

	verifyRequest := &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuth,
		AccessToken: signed.GetToken(),
	}

	res, err := verifier.VerifyClaims(t.Context(), verifyRequest)
	require.NoError(t, err)
	require.Equal(t, &c, res)
}
