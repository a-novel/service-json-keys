package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	testutils "github.com/a-novel/service-json-keys/v2/internal/config"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/models/migrations"
)

func TestJwkInsert(t *testing.T) {
	testCases := []struct {
		name string

		request *dao.JwkInsertRequest
	}{
		{
			name: "WithPublicKey",

			request: &dao.JwkInsertRequest{
				ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				PrivateKey: "cHJpdmF0ZS1rZXktMQ",
				PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
				Usage:      "test-usage",
				Now:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				Expiration: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "WithoutPublicKey",

			request: &dao.JwkInsertRequest{
				ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				PrivateKey: "cHJpdmF0ZS1rZXktMQ",
				Usage:      "test-usage",
				Now:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				Expiration: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	repository := dao.NewJwkInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				testutils.PostgresPresetTest,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					key, err := repository.Exec(ctx, testCase.request)
					require.NoError(t, err)
					require.NotNil(t, key)
				},
			)
		})
	}
}
