UPDATE keys
SET
  deleted_at = ?0,
  -- This operation is the result of a premature suppression,
  -- so a comment should indicate why this happened.
  deleted_comment = ?1
WHERE
  id = ?2
  -- Don't delete already deleted keys.
  AND deleted_at IS NULL
  -- Expired keys cannot be deleted.
  AND expires_at > CURRENT_TIMESTAMP
RETURNING
  *;
