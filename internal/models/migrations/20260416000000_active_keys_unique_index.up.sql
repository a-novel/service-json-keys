-- Adds a unique index on active_keys.id so that the pg_cron job can use
-- REFRESH MATERIALIZED VIEW CONCURRENTLY without locking out reads.
-- PostgreSQL requires a unique index on any materialized view refreshed concurrently;
-- the view inherits no constraints from the underlying keys table.
CREATE UNIQUE INDEX active_keys_id_idx ON active_keys (id);
