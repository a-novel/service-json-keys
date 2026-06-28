# Contributing to service-json-keys

The shared architecture, layers, and conventions live in the [service & architecture concepts](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md); this file covers only what's specific to the JSON Keys service. Platform setup and day-to-day commands are in the [developer onboarding guide](https://github.com/a-novel-kit/.github/blob/master/README.md).

Read the [README](./README.md) first — it covers what the service does and how operators run it.

---

## Quick local interactions

Once the service is up (`a-novel run start service-json-keys/grpc` and/or `.../rest`), the gRPC server listens on `${GRPC_PORT}` and the REST server on `${REST_PORT}`. Both ports are allocated by the `a-novel` daemon; inject them into your shell with `eval "$(a-novel run env service-json-keys)"`.

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

# gRPC: same operations through the private API
grpcurl -plaintext -d '{"usage":"auth"}' localhost:${GRPC_PORT} JwkListService/JwkList
grpcurl -plaintext -d '{"id":"<key-uuid>"}' localhost:${GRPC_PORT} JwkGetService/JwkGet
```

### Signing claims (gRPC only)

Signing requires the master key (`APP_MASTER_KEY`) and is only exposed over the private gRPC API.

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

Adding a usage means updating [`internal/config/jwks.config.yaml`](./internal/config/jwks.config.yaml) in this repo so the new usage is part of the embedded preset. `pkg/go.NewClient` reads `JwkPresetDefault` at startup, so downstream consumers do not add duplicate per-usage config locally; they need a released client-package version that includes the new usage (and, if needed, a new exported `KeyUsageAuth`-style constant) and then upgrade to it.

### Key rotation

[`cmd/rotate-keys/main.go`](./cmd/rotate-keys/main.go) is a one-shot job. For each configured usage, it generates a new key when the current main key is older than `key.rotation`, then refreshes the `active_keys` materialized view so consumers see the change immediately. Run it on a schedule (cron, Kubernetes CronJob, etc.).

This job is **not optional** for a long-running deployment. Existing keys age out of `active_keys` once they reach `key.ttl`, but nothing inside the gRPC or REST processes generates replacements — so without the job firing on schedule, the active set eventually empties for each usage and signing breaks. Run the job once during deploy/bootstrap as well so the database is seeded before the service is expected to sign anything; otherwise it starts with no keys to sign with. Standalone images do this automatically before starting the server (see `builds/standalone.*.Dockerfile`), but split gRPC/REST deployments must arrange that initial run themselves.

### APIs

| API               | Audience                       | Operations                                                              | Spec                                                 |
| ----------------- | ------------------------------ | ----------------------------------------------------------------------- | ---------------------------------------------------- |
| gRPC (`cmd/grpc`) | Internal, private network only | `StatusService`, `JwkGetService`, `JwkListService`, `ClaimsSignService` | [`internal/models/proto/`](./internal/models/proto/) |
| REST (`cmd/rest`) | Public, unauthenticated        | `/ping`, `/healthcheck`, `/jwks`, `/jwk`                                | [`openapi.yaml`](./openapi.yaml)                     |

The REST server never exposes private keys or signing operations. The split is enforced structurally by registering the signing handler only inside [`cmd/grpc/main.go`](./cmd/grpc/main.go). The gRPC server itself implements no application-layer authentication — access control on that server is enforced entirely by deployment infrastructure (network policy, ingress, service mesh).

---

## Questions?

[Open an issue](https://github.com/a-novel/service-json-keys/issues) — include logs and environment details.
