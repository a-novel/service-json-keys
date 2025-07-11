INSERT INTO keys (
  id,
  private_key,
  public_key,
  usage,
  created_at,
  expires_at
) VALUES (
  ?0,
  ?1,
  ?2,
  ?3,
  ?4,
  ?5
) RETURNING *;
