# Contributing to service-json-keys

For platform-wide setup (Go, Node, Podman), the standard `make` targets, and lint/test conventions, see the [generic A-Novel contribution guidelines](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md). This file documents what is specific to the JSON Keys service.

For deployment, configuration, and client-package integration, read the [README](./README.md) first. Contributors are expected to know what the service does and how operators run it before touching the code.

---

## Quick local interactions

Once `make run` is up, the gRPC server listens on `${GRPC_PORT}` and the REST server on `${REST_PORT}`.

### Health

```bash
# REST: liveness
curl http://localhost:${REST_PORT}/ping

# REST: dependency check (Postgres ping)
curl http://localhost:${REST_PORT}/healthcheck

# gRPC: dependency check
grpcurl -plaintext localhost:${GRPC_PORT} StatusService/Status
```

### Reading keys

```bash
# REST: list active public keys for a usage
curl "http://localhost:${REST_PORT}/jwks?usage=auth"

# REST: fetch a single public key by ID
curl "http://localhost:${REST_PORT}/jwk?id=<key-uuid>"

# gRPC: same operations through the private surface
grpcurl -plaintext -d '{"usage":"auth"}' localhost:${GRPC_PORT} JwkListService/JwkList
grpcurl -plaintext -d '{"id":"<key-uuid>"}' localhost:${GRPC_PORT} JwkGetService/JwkGet
```

### Signing claims (gRPC only)

Signing requires the master key (`APP_MASTER_KEY`) and is only exposed over the private gRPC surface.

```bash
grpcurl -plaintext \
  -d '{"usage":"auth","payload":{"@type":"type.googleapis.com/google.protobuf.Struct","value":{"userID":"user-1"}}}' \
  localhost:${GRPC_PORT} \
  ClaimsSignService/ClaimsSign
```

---

## Service-specific concepts

### Master key encryption

Private JWKs are stored encrypted with the application **master key**. The implementation lives in [`internal/lib/masterKeyCrypt.go`](./internal/lib/masterKeyCrypt.go) and uses [NaCl secretbox](https://nacl.cr.yp.to/secretbox.html) — XSalsa20-Poly1305 authenticated encryption with a per-message random 24-byte nonce prepended to the ciphertext.

The master key is loaded from the `APP_MASTER_KEY` env var as a hex-encoded 32-byte secret, parsed by `lib.NewMasterKeyContext`, and pulled out via `lib.MasterKeyContext` on every read or write of a private key payload.

> **Rotating `APP_MASTER_KEY` permanently invalidates every existing private key.** New keys can be re-issued via the rotation job, but legacy keys encrypted under the old master key cannot be decrypted by the new one. This is why the README warns operators not to rotate it.

### JWK lifecycle and the active view

Each usage has at most one **main** key (the latest by `created_at`) and zero or more **legacy** keys (older versions still within their TTL). Producers sign only with the main key; recipients accept tokens signed by any active key for the usage, so a rolling rotation is non-disruptive for token consumers.

The `keys` table — defined in [`internal/models/migrations/`](./internal/models/migrations/) and modelled by `dao.Jwk` in [`internal/dao/pg.jwk.go`](./internal/dao/pg.jwk.go) — records:

| Column                          | Meaning                                                            |
| ------------------------------- | ------------------------------------------------------------------ |
| `created_at`                    | When the key was generated.                                        |
| `expires_at`                    | Hard expiry; the key leaves the active view at this point.         |
| `deleted_at`, `deleted_comment` | Premature revocation (e.g., compromise). `nil` for natural expiry. |

Reads target the `active_keys` materialized view, which excludes expired and soft-deleted rows. The view is refreshed by the rotation job after a write so consumers see the new main key on their next fetch.

### Key configuration

The per-usage configuration ships in [`internal/config/jwks.config.yaml`](./internal/config/jwks.config.yaml). Each top-level key is a usage name; the schema below matches `config.Jwk` in [`internal/config/jwks.config.go`](./internal/config/jwks.config.go):

```yaml
auth:
  alg: EdDSA # signing algorithm: HS256/384/512, ES256/384/512, RS256/384/512, PS256/384/512, EdDSA
  key:
    ttl: 168h # how long a key version stays active before expiring
    rotation: 24h # cadence at which a new key is generated; should be << ttl
    cache: 30m # how long consumers cache fetched public keys before re-fetching
  token:
    ttl: 24h # how long a signed token is valid
    issuer: "..." # JWT iss claim
    audience: "..." # JWT aud claim
    subject: "..." # JWT sub claim
    leeway: 5m # clock-skew tolerance when validating expiry
```

Adding a usage requires the same entry in every consumer that uses `pkg/go` — the `KeyUsageAuth`-style constants pin the usage by name, but the consumer also needs to know its config to wire the matching verifier.

### Key rotation

[`cmd/rotate-keys/main.go`](./cmd/rotate-keys/main.go) is a one-shot job. For each configured usage, it generates a new key when the current main key is older than `key.rotation`, then refreshes the `active_keys` materialized view so consumers see the change immediately. Run it on a schedule (cron, Kubernetes CronJob, etc.). Without it, keys still rotate by TTL — the job just keeps the rotation cadence tight.

### Surfaces

| Surface           | Audience                       | Operations                                                              | Spec                                                 |
| ----------------- | ------------------------------ | ----------------------------------------------------------------------- | ---------------------------------------------------- |
| gRPC (`cmd/grpc`) | Internal, private network only | `StatusService`, `JwkGetService`, `JwkListService`, `ClaimsSignService` | [`internal/models/proto/`](./internal/models/proto/) |
| REST (`cmd/rest`) | Public, unauthenticated        | `/ping`, `/healthcheck`, `/jwks`, `/jwk`                                | [`openapi.yaml`](./openapi.yaml)                     |

The REST surface never exposes private keys or signing operations. The split is enforced structurally by registering the signing handler only inside [`cmd/grpc/main.go`](./cmd/grpc/main.go). The gRPC server itself implements no application-layer authentication — access control on that surface is enforced entirely by deployment infrastructure (network policy, ingress, service mesh).

---

## Questions?

- Open an issue at https://github.com/a-novel/service-json-keys/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
