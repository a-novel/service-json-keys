package pkg_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	golibproto "github.com/a-novel/golib/grpcf/proto/gen"

	"github.com/a-novel/service-json-keys/internal/config/env"
	"github.com/a-novel/service-json-keys/pkg"
)

func TestClient(t *testing.T) {
	t.Parallel()

	client, err := pkg.NewClient(env.GrpcTestUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer client.Close()

	_, err = client.UnaryEcho(t.Context(), &golibproto.UnaryEchoRequest{})
	require.NoError(t, err)

	keys, err := client.JwkList(t.Context(), &pkg.JwkListRequest{
		Usage:   pkg.KeyUsageAuth,
		Private: true,
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(keys.GetKeys()), 1)

	key, err := client.JwkGet(t.Context(), &pkg.JwkGetRequest{
		Id:      keys.GetKeys()[0].GetKid(),
		Private: true,
	})

	require.NoError(t, err)
	require.Equal(t, key.GetJwk(), keys.GetKeys()[0])
}
