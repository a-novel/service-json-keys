-- Restores the materialized view, its indexes and the hourly refresh job.
--
-- Reverting reinstates the staleness this migration removed: expiry and revocation stop taking
-- effect until the next refresh.
DROP VIEW IF EXISTS active_keys;

CREATE MATERIALIZED VIEW active_keys AS (
  SELECT
    *
  FROM
    keys
  WHERE
    expires_at > CURRENT_TIMESTAMP
    AND (
      deleted_at IS NULL
      OR deleted_at > CURRENT_TIMESTAMP
    )
);

CREATE INDEX active_keys_usage_idx ON active_keys (usage);

-- Required by REFRESH MATERIALIZED VIEW CONCURRENTLY.
CREATE UNIQUE INDEX active_keys_id_idx ON active_keys (id);

--bun:split
-- Populate before scheduling, so the first reader does not see an empty view.
REFRESH MATERIALIZED VIEW active_keys;

--bun:split
SELECT
  cron.schedule (
    'refresh-active-keys',
    '0 * * * *',
    $$REFRESH MATERIALIZED VIEW CONCURRENTLY active_keys;$$
  );
