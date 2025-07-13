SELECT cron.unschedule('refresh-active-keys');

DROP EXTENSION IF EXISTS pg_cron;

--bun:split

DROP INDEX IF EXISTS active_keys_usage_idx;

DROP MATERIALIZED VIEW IF EXISTS active_keys;

--bun:split

CREATE VIEW active_keys AS
(
  SELECT *
  FROM keys
  WHERE COALESCE(deleted_at, expires_at) > CURRENT_TIMESTAMP(0)
  ORDER BY id DESC
);
