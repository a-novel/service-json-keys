-- Baseline schema, squashed from the four migrations that preceded it.
--
-- Those four ended at exactly this state, so the history carried no information the schema below
-- does not — only a dependency: one of them scheduled a pg_cron job, which forced every database
-- to have the extension installed purely so the history could replay. Squashing removes the last
-- reference to pg_cron and lets the database image drop it.
--
-- Safe to do here only because nothing is deployed. A database that already ran the old set has no
-- row for this version, so it will try to apply it and fail on the existing table — loudly, which
-- is the intent. Recreate such a database rather than patching around it.
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

Both conditions are required. Testing COALESCE(deleted_at, expires_at) instead would let a
deleted_at in the future act as a backdoor expiry for a key whose expires_at has already passed.

It is a plain view, so the predicates are evaluated per query: a key stops being served the moment
its expires_at passes or its deleted_at is set. Materializing it froze both into a periodic
snapshot, which meant revocation did not take effect until the next refresh. */
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
