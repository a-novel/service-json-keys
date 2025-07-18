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

func TestDeleteKey(t *testing.T) {
	t.Parallel()

	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	now := time.Now().UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		data     dao.DeleteKeyData
		fixtures []*dao.KeyEntity

		expect    *dao.KeyEntity
		expectErr error
	}{
		{
			name: "Success",

			data: dao.DeleteKeyData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},

			expect: &dao.KeyEntity{
				ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				PrivateKey:     "cHJpdmF0ZS1rZXktMg",
				PublicKey:      lo.ToPtr("cHVibGljLWtleS0y"),
				Usage:          models.KeyUsageAuth,
				CreatedAt:      hourAgo,
				ExpiresAt:      hourLater,
				DeletedAt:      &now,
				DeletedComment: lo.ToPtr("foo"),
			},
		},
		{
			name: "NotFound",

			data: dao.DeleteKeyData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},

			expectErr: dao.ErrKeyNotFound,
		},
		{
			name: "AlreadyExpired",

			data: dao.DeleteKeyData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourAgo,
				},
			},

			expectErr: dao.ErrKeyNotFound,
		},
		{
			name: "DeleteMultipleTimes",

			data: dao.DeleteKeyData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey:     "cHJpdmF0ZS1rZXktMg",
					PublicKey:      lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:          models.KeyUsageAuth,
					CreatedAt:      hourAgo,
					ExpiresAt:      hourLater,
					DeletedAt:      lo.ToPtr(now.Add(-30 * time.Minute)),
					DeletedComment: lo.ToPtr("bar"),
				},
			},

			expectErr: dao.ErrKeyNotFound,
		},
	}

	repository := dao.NewDeleteKeyRepository()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				testutils.PostgresPresetTest,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					db, err := postgres.GetContext(ctx)
					require.NoError(t, err)

					if len(testCase.fixtures) > 0 {
						_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
						require.NoError(t, err)
					}

					_, err = db.NewRaw("REFRESH MATERIALIZED VIEW active_keys;").Exec(ctx)
					require.NoError(t, err)

					key, err := repository.DeleteKey(ctx, testCase.data)
					require.ErrorIs(t, err, testCase.expectErr)
					require.Equal(t, testCase.expect, key)
				},
			)
		})
	}
}
