-- Baseline schema: the keys table and the active_keys view every DAO reads through.
CREATE TABLE keys (
  id uuid PRIMARY KEY NOT NULL,
  /* Encrypted private key in JSON Web Key format, base64url-encoded. */
  private_key text NOT NULL CHECK (private_key <> ''),
  /* Public key in JSON Web Key format, base64url-encoded. Null for symmetric keys. */
  public_key text,
  /* Groups keys that serve the same purpose (e.g., "auth"). */
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

/* active_keys exposes only keys that are still valid, and is the only object DAOs read from.

Both conditions are required. deleted_at and expires_at are independent, and a COALESCE over
the pair lets a future deleted_at mask an expires_at that has already passed.

Being a plain view, the predicates are evaluated per query, so a key stops being served the moment
its expires_at passes or its deleted_at is set. */
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
