package handlers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/internal/handlers"
)

func TestRestHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		withPostgres bool

		expectStatus int
		expectHealth string
	}{
		{
			name:         "Success",
			withPostgres: true,
			expectStatus: http.StatusOK,
			expectHealth: handlers.RestHealthStatusUp,
		},
		{
			name:         "Error/MissingPostgres",
			expectStatus: http.StatusServiceUnavailable,
			expectHealth: handlers.RestHealthStatusDown,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if testCase.withPostgres {
				ctx = newPostgresContext(t)
			}

			recorder := httptest.NewRecorder()

			handlers.NewRestHealth().ServeHTTP(
				recorder, httptest.NewRequestWithContext(ctx, http.MethodGet, "/v2/healthcheck", nil),
			)

			response := recorder.Result()
			require.Equal(t, testCase.expectStatus, response.StatusCode)
			require.Contains(t, response.Header.Get("Content-Type"), "application/json")

			body, err := io.ReadAll(response.Body)
			require.NoError(t, errors.Join(err, response.Body.Close()))

			var report any
			require.NoError(t, json.Unmarshal(body, &report))
			// Exact matching protects the public report against raw dependency errors.
			require.Equal(t, map[string]any{
				"client:postgres": map[string]any{"status": testCase.expectHealth},
			}, report)
		})
	}
}
