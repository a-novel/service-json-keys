-- Reverts active_keys to a plain view so its predicates are evaluated per query.
--
-- As a materialized view the predicates were frozen into an hourly snapshot, so expires_at and
-- deleted_at did not take effect when they fell due — they took effect at the next refresh.
-- Revocation was the sharper half: the snapshot holds a *copy* of deleted_at, so revoking a key
-- left the copy reading NULL and the key kept being served, and kept signing, until the refresh.
-- No predicate a reader could add would have seen it; the stale column was in the snapshot itself.
--
-- The leak-safety of this object has always been the predicate, not the materialization: requiring
-- BOTH conditions is what stops deleted_at acting as a backdoor expiry for a key that was never
-- revoked. That predicate is carried over verbatim. Only its evaluation moves, from refresh time to
-- query time.
--
-- Reads stay cheap: keys is bounded by rotation, keys_usage_idx and the primary key cover both
-- query shapes, and every consumer sits behind an in-process key cache.
SELECT
  cron.unschedule ('refresh-active-keys');

-- Drops active_keys_usage_idx and active_keys_id_idx along with it; a plain view needs neither.
DROP MATERIALIZED VIEW IF EXISTS active_keys;

CREATE VIEW active_keys AS (
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
