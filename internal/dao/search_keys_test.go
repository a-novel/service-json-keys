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
	testutils "github.com/a-novel/service-json-keys/internal/test"
	"github.com/a-novel/service-json-keys/models"
)

func TestSearchKeys(t *testing.T) {
	t.Parallel()

	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		usage    models.KeyUsage
		fixtures []*dao.KeyEntity

		expect    []*dao.KeyEntity
		expectErr error
	}{
		{
			name: "FilterUsage",

			usage: models.KeyUsageAuth,

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageRefresh,
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

			expect: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},
		},
		{
			name: "OrderCreation",

			usage: models.KeyUsageAuth,

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo, // Second
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo.Add(time.Minute), // First
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					PrivateKey: "cHJpdmF0ZS1rZXktMw",
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo.Add(-time.Minute), // Third
					ExpiresAt:  hourLater,
				},
			},

			expect: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo.Add(time.Minute), // First
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo, // Second
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					PrivateKey: "cHJpdmF0ZS1rZXktMw",
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo.Add(-time.Minute), // Third
					ExpiresAt:  hourLater,
				},
			},
		},
		{
			name: "IgnoreExpiredAndDeleted",

			usage: models.KeyUsageAuth,

			fixtures: []*dao.KeyEntity{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      models.KeyUsageAuth,
					CreatedAt:  hourAgo,
					ExpiresAt:  hourAgo,
				},
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey:     "cHJpdmF0ZS1rZXktMg",
					PublicKey:      lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:          models.KeyUsageAuth,
					CreatedAt:      hourAgo,
					DeletedAt:      lo.ToPtr(hourAgo),
					DeletedComment: lo.ToPtr("foo"),
					ExpiresAt:      hourLater,
				},
			},

			expect: []*dao.KeyEntity(nil),
		},
	}

	repository := dao.NewSearchKeysRepository()

	for _, testCase := range testCases {
		testutils.TransactionalTest(t, testCase.name, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			if len(testCase.fixtures) > 0 {
				_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
				require.NoError(t, err)
			}

			key, err := repository.SearchKeys(ctx, testCase.usage)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, key)
		})
	}
}
