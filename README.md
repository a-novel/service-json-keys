# Json Keys service

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-json-keys)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-json-keys)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-json-keys)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-json-keys/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/service-json-keys)](https://goreportcard.com/report/github.com/a-novel/service-json-keys)
[![codecov](https://codecov.io/gh/a-novel/service-json-keys/graph/badge.svg?token=almKepuGQE)](https://codecov.io/gh/a-novel/service-json-keys)

![Coverage graph](https://codecov.io/gh/a-novel/service-json-keys/graphs/sunburst.svg?token=almKepuGQE)

## What it does

JSON Keys is a centralized signing-key manager for the A-Novel platform. Services register named **usages** (`auth`, `auth-refresh`, etc.); each usage has its own signing algorithm, rotation schedule, and JWT claim parameters. The service holds every private key and signs tokens on behalf of callers — private key material never leaves the server. Consumers fetch the matching public keys through a read-only REST endpoint and verify tokens locally, with no extra network round-trip per token.

The service ships two surfaces:

- A **private gRPC API** (signing, key retrieval, status) for internal, private-network service-to-service traffic. Anything that touches private key material lives here. The server itself implements no application-layer authentication; access control is enforced externally — by network policy, ingress, service mesh, or other deployment infrastructure.
- A **public REST API** (public-key fetch, health) for any client that needs to verify tokens.

State lives in a PostgreSQL database; both surfaces are stateless and can run as multiple replicas behind a load balancer.

## Running it

The minimal local setup is one Postgres image plus one service image. Pin both to the same release tag (current: `v2.2.6`).

The example below runs the gRPC server in **standalone** mode — the simplest path to a working signing API. Set `GRPC_PORT` to whichever port you want to expose.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.2.6
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - json-keys-postgres-data:/var/lib/postgresql/

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/standalone-grpc:v2.2.6
    ports: ["${GRPC_PORT}:8080"]
    depends_on:
      postgres-json-keys: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks: [api]

networks:
  api:

volumes:
  json-keys-postgres-data:
```

The four deployment shapes:

| Shape           | When to use                       | Image                                                 |
| --------------- | --------------------------------- | ----------------------------------------------------- |
| Standalone gRPC | Local dev, signing experiments    | `service-json-keys/standalone-grpc`                   |
| Standalone REST | Local dev, public-key access only | `service-json-keys/standalone-rest`                   |
| gRPC            | Production gRPC tier              | `service-json-keys/grpc` (+ `migrations`, `database`) |
| REST            | Production REST tier              | `service-json-keys/rest` (+ `migrations`, `database`) |

> Standalone images run migrations on every startup. Convenient for development but unsuitable for production — every replica restart re-runs the migration job. For production, use the split images plus the dedicated `migrations` image.

<details>
<summary>Production-shape example (split gRPC)</summary>

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.2.6
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - json-keys-postgres-data:/var/lib/postgresql/

  migrations-json-keys:
    image: ghcr.io/a-novel/service-json-keys/migrations:v2.2.6
    depends_on:
      postgres-json-keys: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
    networks: [api]

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/grpc:v2.2.6
    ports: ["${GRPC_PORT}:8080"]
    depends_on:
      postgres-json-keys: { condition: service_healthy }
      migrations-json-keys: { condition: service_completed_successfully }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks: [api]

networks:
  api:

volumes:
  json-keys-postgres-data:
```

The REST equivalent uses `service-json-keys/rest:v2.2.6` instead of `grpc:v2.2.6`.

</details>

### Configuration

Configuration is driven by environment variables.

**Required**

| Name             | Description                                                                                                                                                                                                         | Images                                                                         |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `POSTGRES_DSN`   | PostgreSQL connection string.                                                                                                                                                                                       | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest`<br/>`migrations` |
| `APP_MASTER_KEY` | 32-byte hex-encoded master key used to encrypt private keys at rest. **Never rotate** unless you can afford to invalidate every existing private key — see [CONTRIBUTING](./CONTRIBUTING.md#master-key-encryption). | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest`                  |

The gRPC surface exposes private-key operations and must run on an isolated, access-controlled network. Access control is enforced by deployment infrastructure (network policy, ingress, service mesh) — the server does not authenticate callers itself.

**Optional — REST tuning**

| Name                          | Description                          | Default          | Images                       |
| ----------------------------- | ------------------------------------ | ---------------- | ---------------------------- |
| `REST_MAX_REQUEST_SIZE`       | Maximum request body size, in bytes. | `2097152` (2MiB) | `standalone-rest`<br/>`rest` |
| `REST_TIMEOUT_READ`           | Read timeout.                        | `15s`            | `standalone-rest`<br/>`rest` |
| `REST_TIMEOUT_READ_HEADER`    | Header read timeout.                 | `3s`             | `standalone-rest`<br/>`rest` |
| `REST_TIMEOUT_WRITE`          | Write timeout.                       | `30s`            | `standalone-rest`<br/>`rest` |
| `REST_TIMEOUT_IDLE`           | Idle keep-alive timeout.             | `60s`            | `standalone-rest`<br/>`rest` |
| `REST_TIMEOUT_REQUEST`        | Per-request timeout.                 | `60s`            | `standalone-rest`<br/>`rest` |
| `REST_CORS_ALLOWED_ORIGINS`   | CORS allowed origins.                | `*`              | `standalone-rest`<br/>`rest` |
| `REST_CORS_ALLOWED_HEADERS`   | CORS allowed headers.                | `*`              | `standalone-rest`<br/>`rest` |
| `REST_CORS_ALLOW_CREDENTIALS` | CORS allow-credentials flag.         | `false`          | `standalone-rest`<br/>`rest` |
| `REST_CORS_MAX_AGE`           | CORS max-age (seconds).              | `3600`           | `standalone-rest`<br/>`rest` |

**Optional — Logs and tracing**

OpenTelemetry currently supports two exporters: stdout and Google Cloud.

| Name                | Description                                                                    | Default             | Images                                                        |
| ------------------- | ------------------------------------------------------------------------------ | ------------------- | ------------------------------------------------------------- |
| `OTEL`              | Enable OTel tracing. Use the variables below to pick the exporter.             | `false`             | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |
| `GCLOUD_PROJECT_ID` | Google Cloud project ID. When set, switches the OTel exporter to Google Cloud. |                     | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |
| `APP_NAME`          | Application name attached to traces and logs.                                  | `service-json-keys` | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |

## Using the client packages

Two client packages ship with this service:

- **Go** — talks gRPC. Use this from a backend service that needs to sign tokens or verify them with cached public keys.
- **JavaScript / TypeScript** — talks REST. Use this from a frontend or Node service that only needs public keys for local verification.

Each example below is the **minimum viable call**. The full surface is what your editor's intellisense, [pkg.go.dev](https://pkg.go.dev/github.com/a-novel/service-json-keys/v2), and the [API reference](https://a-novel.github.io/service-json-keys-v2) are for.

### Go (gRPC)

```bash
go get github.com/a-novel/service-json-keys/v2
```

```go
package main

import (
	"context"
	"log"

	"github.com/a-novel-kit/golib/grpcf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

type MyClaims struct {
	UserID string `json:"userID"`
}

func main() {
	ctx := context.Background()

	client, err := servicejsonkeys.NewClient(
		"service-json-keys:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Sign claims under the auth usage.
	payload, err := grpcf.InterfaceToProtoAny(MyClaims{UserID: "user-1"})
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.ClaimsSign(ctx, &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuth,
		Payload: payload,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Verify locally — no extra network call per token.
	verifier := servicejsonkeys.NewClaimsVerifier[MyClaims](client)
	claims, err := verifier.VerifyClaims(ctx, &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuth,
		AccessToken: res.GetToken(),
	})
	if err != nil {
		log.Fatal(err)
	}
	_ = claims // claims.UserID == "user-1"
}
```

### JavaScript / TypeScript (REST)

The package is published to GitHub Packages. GitHub requires a Personal Access Token with `repo` and `read:packages` scopes to install from there, even for public packages — see [this discussion](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193).

Add to `.npmrc` (project root or `$HOME`):

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

Install and use:

```bash
pnpm add @a-novel/service-json-keys-rest
```

```typescript
import { JsonKeysApi, jwkList } from "@a-novel/service-json-keys-rest";

const api = new JsonKeysApi("http://service-json-keys:8080");

// Fetch the active public keys for a usage; cache them client-side and verify locally.
const keys = await jwkList(api, "auth");
```

API reference: [a-novel.github.io/service-json-keys-v2](https://a-novel.github.io/service-json-keys-v2)
