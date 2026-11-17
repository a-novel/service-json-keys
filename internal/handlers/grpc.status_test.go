package handlers_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel/service-json-keys/v2/internal/handlers"
	jsonkeysv2 "github.com/a-novel/service-json-keys/v2/internal/handlers/protogen/anovel/jsonkeys/v2"
)

func TestGrpcStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		withPostgres bool

		expect       *jsonkeysv2.StatusResponse
		expectStatus codes.Code
	}{
		{
			name:         "Success",
			withPostgres: true,
			expect: &jsonkeysv2.StatusResponse{
				Postgres: &jsonkeysv2.DependencyHealth{
					Status: jsonkeysv2.DependencyStatus_DEPENDENCY_STATUS_UP,
				},
			},
			expectStatus: codes.OK,
		},
		{name: "Error/MissingPostgres", expectStatus: codes.Unavailable},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if testCase.withPostgres {
				ctx = newPostgresContext(t)
			}

			response, err := handlers.NewGrpcStatus().Status(ctx, &jsonkeysv2.StatusRequest{})
			result, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, testCase.expectStatus, result.Code(), "got %s (%v)", result.Code(), err)
			require.Equal(t, testCase.expect, response)

			if testCase.expectStatus != codes.OK {
				// The message carries no raw dependency error. The fault keeps that error, so the server
				// span names its type.
				require.Equal(t, "service dependencies unavailable", result.Message())
				require.Error(t, errors.Unwrap(err))
			}
		})
	}
}
