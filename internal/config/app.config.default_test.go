package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/config"
)

func TestLoadApp(t *testing.T) {
	testCases := []struct {
		name string

		restTimeoutRead string

		expectRead time.Duration
		expectErr  string
	}{
		{
			name:            "Success",
			restTimeoutRead: "2s",
			expectRead:      2 * time.Second,
		},
		{
			name:            "Error/InvalidRestTimeoutRead",
			restTimeoutRead: "invalid",
			expectErr:       "TEST_REST_TIMEOUT_READ",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("SERVICE_JSON_KEYS_ENV_PREFIX", "TEST_")
			t.Setenv("TEST_REST_TIMEOUT_READ", testCase.restTimeoutRead)

			appConfig, err := config.LoadApp()
			if testCase.expectErr != "" {
				require.ErrorContains(t, err, testCase.expectErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expectRead, appConfig.Rest.Timeouts.Read)
		})
	}
}
