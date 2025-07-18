package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/migrations"
	"github.com/a-novel/service-json-keys/models"
	testutils "github.com/a-novel/service-json-keys/models/config"
)

func TestInsertKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		insertData dao.InsertKeyData
	}{
		{
			name: "WithPublicKey",

			insertData: dao.InsertKeyData{
				ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				PrivateKey: "cHJpdmF0ZS1rZXktMQ",
				PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
				Usage:      models.KeyUsageAuth,
				Now:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				Expiration: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "WithoutPublicKey",

			insertData: dao.InsertKeyData{
				ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				PrivateKey: "cHJpdmF0ZS1rZXktMQ",
				Usage:      models.KeyUsageAuth,
				Now:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				Expiration: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	repository := dao.NewInsertKeyRepository()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				testutils.PostgresPresetTest,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					key, err := repository.InsertKey(ctx, testCase.insertData)
					require.NoError(t, err)
					require.NotNil(t, key)
				},
			)
		})
	}
}
