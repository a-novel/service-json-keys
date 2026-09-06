package servicejsonkeys_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	golibproto "github.com/a-novel-kit/golib/grpcf/proto/gen"

	"github.com/a-novel/service-json-keys/v2/internal/config/env"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

func TestClient(t *testing.T) {
	t.Parallel()

	t.Run("Success/ImportIgnoresServerEnvironment", func(t *testing.T) {
		t.Parallel()

		moduleRoot, err := filepath.Abs("../..")
		require.NoError(t, err)

		consumerDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "go.mod"), []byte(fmt.Sprintf(`module test/client

go 1.27.1

require github.com/a-novel/service-json-keys/v2 v2.0.0

replace github.com/a-novel/service-json-keys/v2 => %s
`, filepath.ToSlash(moduleRoot))), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(consumerDir, "main.go"), []byte(`package main

import _ "github.com/a-novel/service-json-keys/v2/pkg/go"

func main() {}
`), 0o600))

		command := exec.CommandContext(t.Context(), "go", "run", "-mod=mod", ".")
		command.Dir = consumerDir

		command.Env = append(os.Environ(),
			"GOWORK=off",
			"SERVICE_JSON_KEYS_ENV_PREFIX=",
			"REST_TIMEOUT_READ=invalid",
		)

		output, err := command.CombinedOutput()
		require.NoError(t, err, string(output))
	})

	t.Run("Success/DependencyBoundary", func(t *testing.T) {
		t.Parallel()

		command := exec.CommandContext(t.Context(), "go", "list", "-deps", ".")
		output, err := command.Output()
		require.NoError(t, err)

		forbidden := []string{
			"github.com/a-novel/service-json-keys/v2/internal/config",
			"github.com/a-novel/service-json-keys/v2/internal/core",
			"github.com/a-novel/service-json-keys/v2/internal/dao",
			"github.com/a-novel/service-json-keys/v2/internal/handlers",
		}

		for dependency := range strings.SplitSeq(string(output), "\n") {
			for _, prefix := range forbidden {
				require.False(t, strings.HasPrefix(dependency, prefix), dependency)
			}
		}
	})

	t.Run("Success/RemoteCalls", func(t *testing.T) {
		t.Parallel()

		client, err := servicejsonkeys.NewClient(
			env.GrpcUrl,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		require.NoError(t, err)

		defer client.Close()

		_, err = client.UnaryEcho(t.Context(), &golibproto.UnaryEchoRequest{})
		require.NoError(t, err)

		health, err := client.Status(t.Context(), &servicejsonkeys.StatusRequest{})
		require.NoError(t, err)
		require.NotNil(t, health.GetPostgres())

		keys, err := client.JwkList(t.Context(), &servicejsonkeys.JwkListRequest{
			Usage: servicejsonkeys.KeyUsageAuth,
		})
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(keys.GetKeys()), 1)

		key, err := client.JwkGet(t.Context(), &servicejsonkeys.JwkGetRequest{
			Id: keys.GetKeys()[0].GetKid(),
		})
		require.NoError(t, err)
		require.Equal(t, key.GetJwk(), keys.GetKeys()[0])
	})
}
