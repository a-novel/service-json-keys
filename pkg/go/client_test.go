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

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

func main() {
	request := &servicejsonkeys.JwkListRequest{Usage: servicejsonkeys.KeyUsageAuth}
	if request.GetUsage() != servicejsonkeys.KeyUsageAuth {
		panic("client request alias lost its generated methods")
	}

	client, err := servicejsonkeys.NewClient("localhost:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	if _, err := servicejsonkeys.NewClaimsVerifier[map[string]any](client); err != nil {
		panic(err)
	}
}
`), 0o600))

		consumerPath := filepath.Join(consumerDir, "consumer")
		build := exec.CommandContext(t.Context(), "go", "build", "-mod=mod", "-o", "consumer", ".")
		build.Dir = consumerDir

		build.Env = append(os.Environ(), "GOWORK=off")
		output, err := build.CombinedOutput()
		require.NoError(t, err, string(output))

		testCases := []struct {
			name   string
			prefix string
		}{
			{name: "Unprefixed"},
			{name: "Prefixed", prefix: "JSON_KEYS_CLIENT_TEST_"},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				command := exec.CommandContext(t.Context(), consumerPath)
				command.Dir = consumerDir

				command.Env = append(os.Environ(),
					"GOWORK=off",
					"SERVICE_JSON_KEYS_ENV_PREFIX="+testCase.prefix,
					testCase.prefix+"REST_TIMEOUT_READ=invalid",
					testCase.prefix+"GRPC_PING=invalid",
					testCase.prefix+"POSTGRES_PORT=invalid",
					testCase.prefix+"OTEL=invalid",
				)

				output, err := command.CombinedOutput()
				require.NoError(t, err, string(output))
			})
		}
	})

	t.Run("Success/DependencyBoundary", func(t *testing.T) {
		t.Parallel()

		command := exec.CommandContext(t.Context(), "go", "list", "-deps", ".")
		output, err := command.Output()
		require.NoError(t, err)

		dependencies := strings.Split(strings.TrimSpace(string(output)), "\n")

		const configPath = "github.com/a-novel/service-json-keys/v2/internal/config"

		require.NotContains(t, dependencies, configPath)
		require.NotContains(t, dependencies, configPath+"/env")
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
