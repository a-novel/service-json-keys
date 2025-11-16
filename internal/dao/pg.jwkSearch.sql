SELECT
  *
FROM
  active_keys
WHERE
  usage = ?0
ORDER BY
  -- Make sure the main key is returned first.
  created_at DESC
LIMIT
-- No pagination needed, as this limit should never be reached.
  ?1;
