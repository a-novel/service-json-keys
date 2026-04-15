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

	client, err := servicejsonkeys.NewClient(env.GrpcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer client.Close()

	type claims struct {
		Foo string `json:"foo"`
	}

	c := claims{
		Foo: "bar",
	}

	cp, err := grpcf.InterfaceToProtoAny(c)
	require.NoError(t, err)

	signed, err := client.ClaimsSign(t.Context(), &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuth,
		Payload: cp,
	})
	require.NoError(t, err)
	require.NotEmpty(t, signed)

	verifier := servicejsonkeys.NewClaimsVerifier[claims](client)

	res, err := verifier.VerifyClaims(t.Context(), &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuth,
		AccessToken: signed.GetToken(),
	})
	require.NoError(t, err)
	require.Equal(t, &c, res)
}
