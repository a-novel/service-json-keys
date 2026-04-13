-- Converts active_keys from a plain view to a materialized view for read performance,
-- and replaces the COALESCE-based filter with explicit conditions so that deleted_at
-- can no longer be used as a backdoor expiry for keys that have not been revoked.
DROP VIEW IF EXISTS active_keys;

--bun:split
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

--bun:split
REFRESH MATERIALIZED VIEW active_keys;

--bun:split
-- Schedule an hourly background refresh via pg_cron so the materialized view stays current
-- between manual refreshes (which also happen after each key rotation job).
SELECT
  cron.schedule (
    'refresh-active-keys',
    '0 * * * *',
    $$REFRESH MATERIALIZED VIEW CONCURRENTLY active_keys;$$
  );
