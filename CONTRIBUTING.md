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

Once the service is running (`make run`), you can interact with it using `grpcurl` or any gRPC client.

```bash
# List all available methods.
grpcurl -plaintext localhost:4002 list
```

#### Health Checks

```bash
# Simple ping (is the server up?)
grpcurl -plaintext localhost:4002 grpc.health.v1.Health/Check

# Check the status of all services.
grpcurl -plaintext localhost:4002 StatusService/Status
```

#### Key Operations

List available keys:

```bash
grpcurl -plaintext -d '{"usage": "auth"}' localhost:4002 JwkListService/JwkList
```

Get a specific key:

```bash
grpcurl -plaintext -d '{"id": "<key-uuid>"}' localhost:4002 JwkGetService/JwkGet
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
