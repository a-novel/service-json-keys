# Contributing to service-json-keys

This document covers what is specific to the JSON Keys service. Read the [README](./README.md) first for what the service does. The [developer guide](https://a-novel-kit.github.io/stack/) sets up the platform, and the [taxonomy](https://github.com/a-novel/.github/blob/master/docs/taxonomy.md#inside-a-service-layers-and-contracts) explains the layers every service shares.

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

Read an API key by its ID, the `kid` of a token it signed. The local stack holds none, so both calls answer not found. To try one, add `API_KEYS_PUBLISH_NARRATIVE_ENGINE_PLUGIN` to the `environment` of both servers in [`builds/podman-compose.yaml`](./builds/podman-compose.yaml), without committing it:

```bash
API_KEY_ID="paste-a-kid-from-a-token"
curl "http://localhost:${REST_PORT}/v2/api-key?id=${API_KEY_ID}"
grpcurl -plaintext -d "{\"id\":\"${API_KEY_ID}\"}" "localhost:${GRPC_PORT}" anovel.jsonkeys.v2.APIKeyGetService/APIKeyGet
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

`APP_MASTER_KEY` holds the master key as a hex-encoded 32-byte secret. `lib.ParseMasterKey` parses it once at startup, and each command hands the key to the core operations that encrypt or decrypt a private key.

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

A cached key outlives its revocation. The gRPC server signs with it, and verifiers accept it, until their cache refreshes: up to `key.cache` later, or `key.staleIfError` past that while the database is unreachable. Restart the gRPC server and the verifying services to drop a compromised key at once. A revoked main key leaves the usage signing with its newest legacy key, so run the rotation job to give it a new one.

### Key configuration

Each top-level key of [`internal/config/jwkconfig/jwks.config.yaml`](./internal/config/jwkconfig/jwks.config.yaml) is a usage. [`jwkconfig.Usage`](./internal/config/jwkconfig/jwks.config.go) documents every field. A usage signs with Ed25519 (or its legacy label EdDSA), ES256/384/512, RS256/384/512, PS256/384/512 or ML-DSA-44/65/87.

To add a usage, add it to that file. The Go client embeds the same file, so consumers get the usage by upgrading to a client release that has it. Add a `KeyUsage` constant to [`pkg/go/claims.go`](./pkg/go/claims.go) when consumers need to name it.

### Key rotation

[`cmd/rotate-keys`](./cmd/rotate-keys/main.go) is a one-shot job. For each usage, it generates a new key once the main key is older than `key.rotation`, and consumers see the new key on their next fetch. The servers never generate keys, so a deployment schedules the job and runs it once before taking traffic, as the [README](./README.md#deploying) says. The standalone images run it at every boot.

### System API keys

A system API key is an Ed25519 key pair of the platform, made for one purpose. The service holds its public half alone, in the setting named for the purpose, and [`core.APIKeySelect`](./internal/core/apiKeySelect.go) serves it by ID. The ID is the JWK thumbprint of the public key (RFC 7638), so a setting holds nothing but keys.

The first key of a setting is active. Every other one is deactivated: it authenticates no request, and what it signed still verifies. A leaked key leaves the setting, and what it signed stops verifying. A key pair is generated by hand, and the private half never reaches the service.

The Go client verifies what a key signed, on the path that verifies the service's own tokens: [`tokens.APIKeyRequestVerify`](./pkg/go/internal/tokens/apiKeyRequestVerify.go) for a request token and [`tokens.APIKeyProofVerify`](./pkg/go/internal/tokens/apiKeyProofVerify.go) for a proof. Both read the key a token names from the service and keep it for five minutes. No other service or library holds a key or a check of its own.

To add a purpose, add its setting to [`internal/config/env/env.go`](./internal/config/env/env.go) and its `APIKeyPurpose` constant to [`pkg/go/apiKey.go`](./pkg/go/apiKey.go).

### APIs

| API               | Audience                       | Operations                                                          | Spec                                                                                       |
| ----------------- | ------------------------------ | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| gRPC (`cmd/grpc`) | Internal, allowed callers only | `anovel.jsonkeys.v2` services                                       | [`internal/models/proto/anovel/jsonkeys/v2/`](./internal/models/proto/anovel/jsonkeys/v2/) |
| REST (`cmd/rest`) | Public, unauthenticated        | `/v2/ping`, `/v2/healthcheck`, `/v2/jwks`, `/v2/jwk`, `/v2/api-key` | [`openapi.yaml`](./openapi.yaml)                                                           |

Only [`cmd/grpc/main.go`](./cmd/grpc/main.go) loads the master key and registers the signing handler, so the REST server can neither sign nor decrypt a private key.

---

## Code license

The copyrightable source code that the Licensor owns in this repository is released under [RPL-1.5](LICENSE). [NOTICE.md](NOTICE.md) names the Licensor and holds the License Notice and the licensing history.

## Contributor agreement

Before submitting a contribution for inclusion in an official A-Novel repository, review the [Contributor License Agreement](CLA.md). The Project Owner may require a signed individual or corporate agreement before accepting the contribution. A pull request, a checkbox or a signed-off commit does not replace that signature.

If you developed the contribution for an employer or a client, or it includes third-party material, make sure you may grant the rights in the agreement, and disclose any third-party license terms.

The agreement gives the Project Owner additional rights over accepted contributions, including proprietary relicensing where lawful. It changes neither the RPL-1.5 license of public releases nor anything for the maintainers of independent forks who have not signed it.

## Submitting changes

1. Open an issue or pull request describing the proposed change.
2. Identify any incorporated third-party material and its license notices.
3. When asked, sign the agreement and identify your GitHub account, the repository and the pull request or commit it covers.
4. Keep existing copyright and license notices intact.

The Project Owner records signatures and matches them to accepted pull requests or commits. GitHub neither executes nor validates the agreement.

---

## Questions?

[Open an issue](https://github.com/a-novel/service-json-keys/issues) with the logs and environment details.
