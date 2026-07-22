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

// TestPgJwkDeleteTakesEffectImmediately pins that a revoked key stops being served from the
// moment it is revoked, with no refresh in between.
//
// It reads through PgJwkSelect, because the guarantee lives on the read path: active_keys is a
// plain view, so its predicates are evaluated per query and a revocation takes effect at once.
func TestPgJwkDeleteTakesEffectImmediately(t *testing.T) {
	t.Parallel()

	// Truncate, never Round: this value becomes deleted_at, and the view keeps a key
	// visible while deleted_at > CURRENT_TIMESTAMP. Round goes to the nearest second, so
	// half the time it lands in the future and the key stays active for up to 500ms —
	// which the assertion below then reads as "revocation did not take effect".
	now := time.Now().UTC().Truncate(time.Second)
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

			// Precondition: the key is served before revocation.
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
