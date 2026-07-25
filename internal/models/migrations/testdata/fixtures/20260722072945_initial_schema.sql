-- Rows span every active_keys predicate outcome.
INSERT INTO
  keys (
    id,
    private_key,
    public_key,
    usage,
    created_at,
    expires_at,
    deleted_at,
    deleted_comment
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    'fixture-private-active',
    NULL,
    'roundtrip',
    '2026-07-22T07:29:45Z',
    '2099-01-01T00:00:00Z',
    NULL,
    NULL
  ),
  (
    '00000000-0000-0000-0000-000000000002',
    'fixture-private-expired',
    NULL,
    'roundtrip',
    '2020-01-01T00:00:00Z',
    '2021-01-01T00:00:00Z',
    NULL,
    NULL
  ),
  (
    '00000000-0000-0000-0000-000000000003',
    'fixture-private-revoked',
    NULL,
    'roundtrip',
    '2026-07-22T07:29:45Z',
    '2099-01-01T00:00:00Z',
    '2026-07-23T07:29:45Z',
    'fixture revocation'
  );
