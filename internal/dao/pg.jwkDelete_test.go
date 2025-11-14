package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	testutils "github.com/a-novel/service-json-keys/internal/config"
	"github.com/a-novel/service-json-keys/internal/dao"
	"github.com/a-novel/service-json-keys/internal/models/migrations"
)

func TestJwkDelete(t *testing.T) {
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	now := time.Now().UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		request  *dao.JwkDeleteRequest
		fixtures []*dao.Jwk

		expect    *dao.Jwk
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.JwkDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
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

			expect: &dao.Jwk{
				ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				PrivateKey:     "cHJpdmF0ZS1rZXktMg",
				PublicKey:      lo.ToPtr("cHVibGljLWtleS0y"),
				Usage:          "test-usage",
				CreatedAt:      hourAgo,
				ExpiresAt:      hourLater,
				DeletedAt:      &now,
				DeletedComment: lo.ToPtr("foo"),
			},
		},
		{
			name: "NotFound",

			request: &dao.JwkDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
			},

			expectErr: dao.ErrJwkDeleteNotFound,
		},
		{
			name: "AlreadyExpired",

			request: &dao.JwkDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey: "cHJpdmF0ZS1rZXktMg",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourAgo,
				},
			},

			expectErr: dao.ErrJwkDeleteNotFound,
		},
		{
			name: "DeleteMultipleTimes",

			request: &dao.JwkDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.Jwk{
				{
					ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					PrivateKey: "cHJpdmF0ZS1rZXktMQ",
					PublicKey:  lo.ToPtr("cHVibGljLWtleS0x"),
					Usage:      "test-usage",
					CreatedAt:  hourAgo,
					ExpiresAt:  hourLater,
				},
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					PrivateKey:     "cHJpdmF0ZS1rZXktMg",
					PublicKey:      lo.ToPtr("cHVibGljLWtleS0y"),
					Usage:          "test-usage",
					CreatedAt:      hourAgo,
					ExpiresAt:      hourLater,
					DeletedAt:      lo.ToPtr(now.Add(-30 * time.Minute)),
					DeletedComment: lo.ToPtr("bar"),
				},
			},

			expectErr: dao.ErrJwkDeleteNotFound,
		},
	}

	repository := dao.NewJwkDelete()

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
