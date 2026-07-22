package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-json-keys/v2/internal/config/configtest"
	"github.com/a-novel/service-json-keys/v2/internal/dao"
	"github.com/a-novel/service-json-keys/v2/internal/models/migrations"
)

func TestPgJwkDelete(t *testing.T) {
	t.Parallel()

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
			name: "Error/NotFound",

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
			name: "Error/AlreadyExpired",

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
			name: "Error/AlreadyDeleted",

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

	dao := dao.NewPgJwkDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunDBTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					db, err := postgres.GetContext(ctx)
					require.NoError(t, err)

					if len(testCase.fixtures) > 0 {
						_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
						require.NoError(t, err)
					}

					key, err := dao.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)
					require.Equal(t, testCase.expect, key)
				},
			)
		})
	}
}

// TestPgJwkDeleteTakesEffectImmediately is the regression this milestone's issue is about: a
// revoked key must stop being served from the moment it is revoked, with no refresh in between.
//
// It reads through PgJwkSelect rather than asserting on the delete's own return value, because
// the defect lived entirely on the read path. While active_keys was a materialized view the
// snapshot carried a *copy* of deleted_at, so the revoked key kept being returned — and kept
// signing — until the next hourly refresh. Nothing the reader could filter on would have caught
// it, since the stale column was in the snapshot itself.
func TestPgJwkDeleteTakesEffectImmediately(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	keyID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	postgres.RunDBTest(
		t,
		configtest.PostgresPreset,
		migrations.Migrations,
		func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			fixture := &dao.Jwk{
				ID:         keyID,
				PrivateKey: "private-key",
				PublicKey:  lo.ToPtr("public-key"),
				Usage:      "auth",
				CreatedAt:  now.Add(-time.Hour),
				ExpiresAt:  now.Add(time.Hour),
			}

			_, err = db.NewInsert().Model(fixture).Exec(ctx)
			require.NoError(t, err)

			selectDAO := dao.NewPgJwkSelect()

			// Precondition: the key is served before revocation. Without this the test would
			// still pass against a read path that returns nothing at all.
			got, err := selectDAO.Exec(ctx, &dao.JwkSelectRequest{ID: keyID})
			require.NoError(t, err)
			require.Equal(t, keyID, got.ID)

			_, err = dao.NewPgJwkDelete().Exec(ctx, &dao.JwkDeleteRequest{
				ID:      keyID,
				Now:     now,
				Comment: "compromised",
			})
			require.NoError(t, err)

			// No refresh here on purpose — that is the whole assertion.
			_, err = selectDAO.Exec(ctx, &dao.JwkSelectRequest{ID: keyID})
			require.ErrorIs(t, err, dao.ErrJwkSelectNotFound)
		},
	)
}
