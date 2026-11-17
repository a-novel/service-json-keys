package config_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppPresetDefault(t *testing.T) {
	t.Parallel()

	// The preset resolves at package initialization, so the test reads it in a process of its own.
	// The unprefixed variable is invalid, which proves the prefix is applied.
	command := exec.CommandContext(t.Context(), "go", "run", "./testdata/app")

	command.Env = append(os.Environ(),
		"GOWORK=off",
		"SERVICE_JSON_KEYS_ENV_PREFIX=TEST_",
		"REST_TIMEOUT_READ=invalid",
		"TEST_REST_TIMEOUT_READ=2s",
	)

	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
	require.Equal(t, "2s", strings.TrimSpace(string(output)))
}
