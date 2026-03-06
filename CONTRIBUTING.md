# Contributing to service-json-keys

Welcome to the JSON Keys service for the A-Novel platform. This guide will help you understand the codebase, set
up your development environment, and contribute effectively.

Before reading this guide, if you haven't already, please check the
[generic contribution guidelines](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md) that are relevant
to your scope.

---

## Quick Start

### Prerequisites

The following must be installed on your system.

- [Go](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download)
  - [pnpm](https://pnpm.io/installation)
- [Podman](https://podman.io/docs/installation)
- (optional) [Direnv](https://direnv.net/)
- Make
  - `sudo apt-get install build-essential` (apt)
  - `sudo pacman -S make` (arch)
  - `brew install make` (macOS)
  - [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

### Bootstrap

Create a `.envrc` file in the project root:

```bash
cp .envrc.template .envrc
```

Then, load the environment variables:

```bash
direnv allow .
# Alternatively, if you don't have direnv on your system
source .envrc
```

Finally, install the dependencies:

```bash
make install
```

### Common Commands

| Command         | Description                      |
| --------------- | -------------------------------- |
| `make run`      | Start all services locally       |
| `make test`     | Run all tests                    |
| `make lint`     | Run all linters                  |
| `make format`   | Format all code                  |
| `make build`    | Build Docker images locally      |
| `make generate` | Generate mocks and protobuf code |

### Interacting with the Service

Once the service is running (`make run`), you can interact with it using:

- `curl` or any HTTP client (REST API).
- `grpcurl` or any gRPC client (gRPC API).

#### Health Checks

```bash
# REST: Simple ping (is the server up?)
curl http://localhost:${SERVICE_JSON_KEYS_REST_PORT}/ping

# REST: Detailed health check (checks database, dependencies)
curl http://localhost:${SERVICE_JSON_KEYS_REST_PORT}/healthcheck

# gRPC: Simple ping (is the server up?)
grpcurl -plaintext localhost:${SERVICE_JSON_KEYS_GRPC_PORT} grpc.health.v1.Health/Check

# gRPC: Check the status of all services.
grpcurl -plaintext localhost:${SERVICE_JSON_KEYS_GRPC_PORT} StatusService/Status
```

#### Key Operations

List available keys:

```bash
# REST
curl "http://localhost:${SERVICE_JSON_KEYS_REST_PORT}/jwks?usage=auth"

# gRPC
grpcurl -plaintext -d '{"usage": "auth"}' "localhost:${SERVICE_JSON_KEYS_GRPC_PORT}" JwkListService/JwkList
```

Get a specific key:

```bash
# REST
curl "http://localhost:${SERVICE_JSON_KEYS_REST_PORT}/jwk?id=<key-uuid>"

# gRPC
grpcurl -plaintext -d '{"id": "<key-uuid>"}' "localhost:${SERVICE_JSON_KEYS_GRPC_PORT}" JwkGetService/JwkGet
```

---

## Project-Specific Guidelines

> This section contains patterns specific to this JSON Keys service.

### Master Key Encryption

Private JWKs are stored encrypted using AES-GCM with a master key. The master key is:

- Loaded from `APP_MASTER_KEY` environment variable
- Stored in context via `lib.NewMasterKeyContext()`
- Used to encrypt/decrypt private key payloads before storage/retrieval

**Critical:** The master key should **never** be rotated unless absolutely necessary — changing it will permanently lose access to all existing encrypted keys.

### JWK Lifecycle

Keys go through the following states:

1. **Generated**: New key created with expiration time
2. **Active**: Key is used for signing
3. **Expired**: Key past expiration, still valid for verification
4. **Deleted**: Key removed from database

### Key Configuration

Key types and their settings are defined in `internal/config/jwks.config.yaml`:

```yaml
auth:
  algorithm: ES256
  rotation: 720h
  expiry: 8760h
```

Each key usage (e.g., `auth`) can have different algorithms, rotation schedules, and expiry times.

### Key Rotation

The `rotate-keys` job (`cmd/rotate-keys/main.go`) handles automatic key rotation:

- Generates new keys when current ones approach expiration
- Deletes keys past their retention period
- Should be run as a scheduled job (cron, Kubernetes CronJob, etc.)

### gRPC Services

| Service             | Purpose                    |
| ------------------- | -------------------------- |
| `StatusService`     | Health and status checks   |
| `JwkGetService`     | Retrieve single key by ID  |
| `JwkListService`    | List keys by usage         |
| `ClaimsSignService` | Sign JWT claims with a key |

### REST API

The REST API serves public JWK data over HTTP. It is documented via an OpenAPI spec:

- `openapi.yaml` — machine-readable spec
- `openapi.html` — interactive Scalar API Reference viewer (open in a browser)

The published API reference is hosted at [GitHub Pages](https://a-novel.github.io/service-json-keys-v2).

### JavaScript Client Package

Frontend or Node.js consumers can use `@a-novel/service-json-keys-rest` (`pkg/rest-js/`) to call the REST API:

```typescript
import { JsonKeysApi, jwkGet, jwkList } from "@a-novel/service-json-keys-rest";

const api = new JsonKeysApi("http://localhost:4021");

// Check service health.
await api.ping();

// Fetch public keys.
const keys = await jwkList(api, "auth");
const key = await jwkGet(api, "<key-id>");
```

Integration tests for the JS client live in `pkg/test/rest-js/`. Run them locally with:

```bash
make test-pkg-js
```

### Go Client Package

Other services integrate with json-keys via the `pkg/` package:

```go
import jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

// Create client
client, err := jkpkg.NewClient("<grpc-address>")

// Sign claims
token, err := client.ClaimsSign(ctx, &jkpkg.ClaimsSignRequest{
    Usage:   "auth",
    Payload: claimsPayload,
})

// Verify claims
verifier := jkpkg.NewClaimsVerifier[MyClaims](client)
claims, err := verifier.VerifyClaims(ctx, &jkpkg.VerifyClaimsRequest{
    Usage:       "auth",
    AccessToken: token.GetToken(),
})
```

---

## Questions?

If you have questions or run into issues:

- Open an issue at https://github.com/a-novel/service-json-keys/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
