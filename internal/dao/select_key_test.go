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

func TestSelectKey(t *testing.T) {
	t.Parallel()

	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		fixtures []*dao.KeyEntity

		id uuid.UUID

		expect    *dao.KeyEntity
		expectErr error
	}{
		{
			name: "Success",

			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

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
				ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				PrivateKey: "cHJpdmF0ZS1rZXktMg",
				PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
				Usage:      models.KeyUsageAuth,
				CreatedAt:  hourAgo,
				ExpiresAt:  hourLater,
			},
		},
		{
			name: "NotFound",

			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

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
			name: "IgnoreExpired",

			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

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
	}

	repository := dao.NewSelectKeyRepository()

	for _, testCase := range testCases {
		testutils.TransactionalTest(t, testCase.name, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			if len(testCase.fixtures) > 0 {
				_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
				require.NoError(t, err)
			}

			key, err := repository.SelectKey(ctx, testCase.id)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, key)
		})
	}
}
