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

func TestJwkSearch(t *testing.T) {
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		request  *dao.JwkSearchRequest
		fixtures []*dao.Jwk

		expect    []*dao.Jwk
		expectErr error
	}{
		{
			name: "FilterUsage",

			request: &dao.JwkSearchRequest{
				Usage: "test-usage",
			},

			fixtures: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "other-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},

			expect: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},
		},
		{
			name: "OrderCreation",

			request: &dao.JwkSearchRequest{
				Usage: "test-usage",
			},

			fixtures: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo, // Second
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo.Add(time.Minute), // First
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					PrivateKey: "cHJpdmF0ZS1rZXktMw",
					Usage:      "test-usage",
					CreatedAt:  hourAgo.Add(-time.Minute), // Third
					ExpiresAt:  hourLater,
				},
			},

			expect: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo.Add(time.Minute), // First
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo, // Second
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					PrivateKey: "cHJpdmF0ZS1rZXktMw",
					Usage:      "test-usage",
					CreatedAt:  hourAgo.Add(-time.Minute), // Third
					ExpiresAt:  hourLater,
				},
			},
		},
		{
			name: "IgnoreExpiredAndDeleted",

			request: &dao.JwkSearchRequest{
				Usage: "test-usage",
			},

			fixtures: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourAgo,
				},
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey:     "cHJpdmF0ZS1rZXktMg",
					PublicKey:      lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:          "test-usage",
					CreatedAt:      hourAgo,
					DeletedAt:      lo.ToPtr(hourAgo),
					DeletedComment: lo.ToPtr("foo"),
					ExpiresAt:      hourLater,
				},
			},

			expect: []*dao.Jwk(nil),
		},
	}

	repository := dao.NewJwkSearch()

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

					key, err := repository.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)
					require.Equal(t, testCase.expect, key)
				},
			)
		})
	}
}
