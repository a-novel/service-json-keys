package dao_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/a-novel/service-json-keys/v2/internal/models/migrations"
)

// cronStubUp creates a no-op `cron` schema so migrations that call
// cron.schedule / cron.unschedule apply against a database that does not have
// pg_cron.
//
// postgres.RunDBTest gives every test its own freshly created database. pg_cron
// is inherently single-database (the extension and its `cron` schema live only
// in the database named by cron.database_name), so a fresh database has no
// `cron` schema and migration 20250713173700 (`cron.schedule(...)` for the
// hourly active_keys refresh) would fail with `schema "cron" does not exist`.
//
// The scheduled job is irrelevant to tests anyway: it never fires during a test
// run, and every DAO test that needs a current active_keys materialized view
// already issues `REFRESH MATERIALIZED VIEW active_keys` itself. So a stub whose
// schedule/unschedule are no-ops is functionally complete — it only has to let
// the migration apply.
const (
	cronStubUp = `CREATE SCHEMA IF NOT EXISTS cron;

CREATE OR REPLACE FUNCTION cron.schedule(text, text, text)
RETURNS bigint LANGUAGE sql AS 'SELECT 0::bigint';

CREATE OR REPLACE FUNCTION cron.unschedule(text)
RETURNS boolean LANGUAGE sql AS 'SELECT true';
`

	cronStubDown = `DROP SCHEMA IF EXISTS cron CASCADE;
`
)

// migrationsWithCronStub returns the embedded migration set with a stub-cron
// migration prepended. Its 0001-01-01 timestamp sorts (bun orders migrations by
// the leading numeric version) before every real migration, so the stub `cron`
// schema exists by the time 20250713173700 runs.
//
// It is materialised as an fstest.MapFS rather than an fs.FS overlay so bun's
// fs.WalkDir-based discovery sees a single, fully-formed directory with no
// custom fs.FS plumbing.
func migrationsWithCronStub(t *testing.T) fs.FS {
	t.Helper()

	merged := fstest.MapFS{}

	err := fs.WalkDir(migrations.Migrations, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		data, err := fs.ReadFile(migrations.Migrations, path)
		if err != nil {
			return err
		}

		merged[path] = &fstest.MapFile{Data: data}

		return nil
	})
	if err != nil {
		panic(err)
	}

	merged["00010101000000_cron_stub.up.sql"] = &fstest.MapFile{Data: []byte(cronStubUp)}
	merged["00010101000000_cron_stub.down.sql"] = &fstest.MapFile{Data: []byte(cronStubDown)}

	return merged
}
