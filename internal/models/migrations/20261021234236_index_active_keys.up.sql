-- Keys stay in the table once they expire, and active_keys filters them out at read time. With
-- expires_at after usage in the index, a search range-scans the unexpired keys of its usage instead
-- of every key the usage ever had. The new index covers the usage index, which it replaces.
CREATE INDEX keys_usage_expires_at_idx ON keys (usage, expires_at);

DROP INDEX keys_usage_idx;
