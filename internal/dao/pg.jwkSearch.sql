SELECT
  *
FROM
  active_keys
WHERE
  usage = ?0
ORDER BY
  created_at DESC
LIMIT
  ?1;
