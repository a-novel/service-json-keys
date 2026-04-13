CREATE TABLE keys (
  id uuid PRIMARY KEY NOT NULL,
  /* Encrypted private key in JSON Web Key format, base64url-encoded. */
  private_key text NOT NULL CHECK (private_key <> ''),
  /* Public key in JSON Web Key format, base64url-encoded. Null for symmetric keys. */
  public_key text,
  /* Groups keys that serve the same signing purpose (e.g., "auth", "auth-refresh"). */
  usage text NOT NULL,
  created_at timestamp(0) with time zone NOT NULL,
  /* Hard expiry date. Once passed, the key is excluded from the active view. */
  expires_at timestamp(0) with time zone NOT NULL,
  /* Soft-delete timestamp. Set when a key is revoked early (e.g., due to a compromise). */
  deleted_at timestamp(0) with time zone,
  /* Human-readable reason for the early revocation. */
  deleted_comment text
);

CREATE INDEX keys_usage_idx ON keys (usage);

/* active_keys exposes only keys that have not yet expired and have not been soft-deleted.
A key is considered active when deleted_at is null (or in the future) AND expires_at is in the future. */
CREATE VIEW active_keys AS (
  SELECT
    *
  FROM
    keys
  WHERE
    COALESCE(deleted_at, expires_at) > CURRENT_TIMESTAMP(0)
  ORDER BY
    id DESC
);
