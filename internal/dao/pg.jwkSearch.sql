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
  -- Safeguard against runaway results; normal rotation keeps the active-key count well below this.
  ?1;
