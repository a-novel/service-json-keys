# Contributing to service-json-keys

This document covers what is specific to the JSON Keys service. Read the [README](./README.md) first for what the service does. The [developer guide](https://a-novel-kit.github.io/stack/) sets up the platform, and the [service concepts](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md) explain the layers every service shares.

---

## Quick local interactions

Start the servers, then load the ports the `a-novel` daemon gave them into your shell:

```bash
a-novel run start service-json-keys/grpc
a-novel run start service-json-keys/rest
eval "$(a-novel run env service-json-keys)"
```

### Health

```bash
# REST: liveness
curl "http://localhost:${REST_PORT}/v2/ping"

# REST: dependency check (Postgres ping)
curl "http://localhost:${REST_PORT}/v2/healthcheck"

# gRPC: dependency check
grpcurl -plaintext "localhost:${GRPC_PORT}" anovel.jsonkeys.v2.StatusService/Status
```

### Reading keys

```bash
# REST: list the active public keys of a usage
curl "http://localhost:${REST_PORT}/v2/jwks?usage=auth"

# gRPC: the same list through the private API
grpcurl -plaintext -d '{"usage":"auth"}' "localhost:${GRPC_PORT}" anovel.jsonkeys.v2.JwkListService/JwkList
```

Fetch one key by the `kid` of a listed key:

```bash
KEY_ID="paste-a-kid-from-the-list"
curl "http://localhost:${REST_PORT}/v2/jwk?id=${KEY_ID}"
grpcurl -plaintext -d "{\"id\":\"${KEY_ID}\"}" "localhost:${GRPC_PORT}" anovel.jsonkeys.v2.JwkGetService/JwkGet
```

### Signing claims

Only the gRPC API signs. The local stack provides a test master key.

```bash
grpcurl -plaintext \
  -d '{"usage":"auth","payload":{"@type":"type.googleapis.com/google.protobuf.Struct","value":{"userID":"user-1"}}}' \
  "localhost:${GRPC_PORT}" \
  anovel.jsonkeys.v2.ClaimsSignService/ClaimsSign
```

---

## Service-specific concepts

### Master key encryption

The service stores each private key encrypted with the **master key**. [`internal/lib/masterKeyCrypt.go`](./internal/lib/masterKeyCrypt.go) uses [NaCl secretbox](https://nacl.cr.yp.to/secretbox.html): XSalsa20-Poly1305 authenticated encryption, with a random 24-byte nonce prepended to each ciphertext.

`APP_MASTER_KEY` holds the master key as a hex-encoded 32-byte secret. `lib.NewMasterKeyContext` parses it into the context, and `lib.MasterKeyContext` reads it back whenever a private key is encrypted or decrypted.

> **Changing `APP_MASTER_KEY` leaves every stored private key unreadable.** Signing reads all the active keys of a usage, so it fails until every key encrypted under the old value leaves the active view.

### JWK lifecycle and the active view

Each usage has at most one **main** key, the latest by `created_at`, and any number of **legacy** keys, older versions still within their TTL. Producers sign only with the main key. Recipients accept a token signed by any active key of the usage, so a rotation does not disrupt them.

[`dao.Jwk`](./internal/dao/pg.jwk.go) models the `keys` table, which the [migrations](./internal/models/migrations/) define. Three columns drive the lifecycle:

| Column                          | Meaning                                                           |
| ------------------------------- | ----------------------------------------------------------------- |
| `created_at`                    | When the key was generated.                                       |
| `expires_at`                    | Hard expiry; the key leaves the active view at this point.        |
| `deleted_at`, `deleted_comment` | Early revocation, after a compromise for example. Null otherwise. |

Reads go through the `active_keys` view, which excludes expired and revoked rows on every query. A key stops being served the moment its `expires_at` or `deleted_at` passes.

### Key configuration

Each top-level key of [`internal/config/jwk/jwks.config.yaml`](./internal/config/jwk/jwks.config.yaml) is a usage. [`jwk.Jwk`](./internal/config/jwk/jwks.config.go) documents every field. A usage signs with Ed25519 (or its legacy label EdDSA), ES256/384/512, RS256/384/512, PS256/384/512 or ML-DSA-44/65/87.

To add a usage, add it to that file. The Go client embeds the same file, so consumers get the usage by upgrading to a client release that has it. Add a `KeyUsage` constant to [`pkg/go/claims.go`](./pkg/go/claims.go) when consumers need to name it.

### Key rotation

[`cmd/rotate-keys`](./cmd/rotate-keys/main.go) is a one-shot job. For each usage, it generates a new key once the main key is older than `key.rotation`, and consumers see the new key on their next fetch. The servers never generate keys, so a deployment schedules the job and runs it once before taking traffic, as the [README](./README.md#deploying) says. The standalone images run it at every boot.

### APIs

| API               | Audience                       | Operations                                           | Spec                                                                                       |
| ----------------- | ------------------------------ | ---------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| gRPC (`cmd/grpc`) | Internal, private network only | `anovel.jsonkeys.v2` services                        | [`internal/models/proto/anovel/jsonkeys/v2/`](./internal/models/proto/anovel/jsonkeys/v2/) |
| REST (`cmd/rest`) | Public, unauthenticated        | `/v2/ping`, `/v2/healthcheck`, `/v2/jwks`, `/v2/jwk` | [`openapi.yaml`](./openapi.yaml)                                                           |

Only [`cmd/grpc/main.go`](./cmd/grpc/main.go) loads the master key and registers the signing handler, so the REST server can neither sign nor decrypt a private key.

---

## Questions?

[Open an issue](https://github.com/a-novel/service-json-keys/issues) with the logs and environment details.
