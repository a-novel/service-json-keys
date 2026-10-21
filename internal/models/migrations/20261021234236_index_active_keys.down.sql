CREATE INDEX IF NOT EXISTS keys_usage_idx ON keys (usage);

DROP INDEX IF EXISTS keys_usage_expires_at_idx;
