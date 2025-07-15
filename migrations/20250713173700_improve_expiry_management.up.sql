DROP VIEW IF EXISTS active_keys;

--bun:split

CREATE MATERIALIZED VIEW active_keys AS
(
  SELECT *
  FROM keys
  WHERE
    expires_at > CURRENT_TIMESTAMP
    AND (deleted_at IS NULL OR deleted_at > CURRENT_TIMESTAMP)
);

CREATE INDEX active_keys_usage_idx ON active_keys (usage);

--bun:split

REFRESH MATERIALIZED VIEW active_keys;

--bun:split

-- Automatically refresh the materialized view.
SELECT cron.schedule(
  'refresh-active-keys',
  '0 * * * *',
  $$REFRESH MATERIALIZED VIEW CONCURRENTLY active_keys;$$
);
