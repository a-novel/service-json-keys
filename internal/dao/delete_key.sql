UPDATE active_keys SET deleted_at = ?0, deleted_comment = ?1
WHERE id = ?2 AND deleted_at IS NULL RETURNING *;
