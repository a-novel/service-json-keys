# JSON Keys service

Centralized signing-key manager for the A-Novel platform: it holds every private key, signs tokens over a private gRPC API, and serves the matching public keys over REST so callers verify locally.

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-json-keys)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-json-keys)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-json-keys)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-json-keys/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/service-json-keys)](https://goreportcard.com/report/github.com/a-novel/service-json-keys)
[![codecov](https://codecov.io/gh/a-novel/service-json-keys/graph/badge.svg)](https://codecov.io/gh/a-novel/service-json-keys)

![Coverage graph](https://codecov.io/gh/a-novel/service-json-keys/graphs/sunburst.svg)

## What it does

Services register named **usages** (`auth`, `auth-refresh`, …), each with its own signing algorithm, rotation schedule, and claim parameters. JSON Keys holds every private key and signs on callers' behalf — key material never leaves the server. Consumers fetch the matching public keys once and verify tokens locally, with no per-token round-trip.

Two APIs:

- **Private gRPC API** — signing, key retrieval, status — for internal service-to-service traffic. Everything touching private keys lives here. The server has no application-layer auth; access control is external (network policy, ingress, service mesh).
- **Public REST API** — public-key fetch, health — for anyone verifying tokens.

## Deploying

The service runs as published OCI images plus a PostgreSQL database. Both servers are stateless, so each scales to as many replicas as you need behind a load balancer; all state lives in Postgres.

> **OpenTofu modules are the planned canonical deployment path.** Until they land, deploy the images with any container orchestrator — the composition below is the reference for which images to run, how they wire together, and the environment they expect.

| Image                               | Role                                                                        |
| ----------------------------------- | --------------------------------------------------------------------------- |
| `service-json-keys/grpc`            | Private signing + key-management API. Internal network only.                |
| `service-json-keys/rest`            | Public key-fetch + health API.                                              |
| `service-json-keys/jobs/migrations` | One-shot schema migration job; runs to completion before the servers start. |
| `service-json-keys/jobs/rotatekeys` | Scheduled key-rotation job (see [Configuration](#configuration)).           |
| `service-json-keys/database`        | Pre-tuned PostgreSQL image — or bring your own Postgres.                    |

Pin every image to the same release tag — see the [latest release](https://github.com/a-novel/service-json-keys/releases/latest). A production deployment runs `database`, then the migrations job to completion, then any number of `grpc` and/or `rest` replicas:

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.3.1
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
    image: ghcr.io/a-novel/service-json-keys/jobs/migrations:v2.3.1
    depends_on:
      postgres-json-keys: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
    networks: [api]

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/grpc:v2.3.1 # or .../rest:v2.3.1 for the public REST API
    ports: ["${GRPC_PORT}:8080"] # the container always listens on 8080; map ${REST_PORT} for the rest image
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

Run both servers by adding a second service that reuses the same database and migrations with the `rest` image. Key rotation is a separate scheduled job — run the `service-json-keys/jobs/rotatekeys` image on a timer (see [CONTRIBUTING](./CONTRIBUTING.md#key-rotation)); without it, active keys eventually age out and signing stops.

### Configuration

Every variable is read from the process environment.

| Name             | Description                                                                                                                                                                                                                                               | Images                                                                              |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `POSTGRES_DSN`   | PostgreSQL connection string. **Required.**                                                                                                                                                                                                               | all                                                                                 |
| `APP_MASTER_KEY` | 32-byte hex-encoded key that encrypts private keys at rest. **Required** by every image that touches private keys. **Never rotate** unless you can afford to invalidate every existing key — see [CONTRIBUTING](./CONTRIBUTING.md#master-key-encryption). | `grpc`<br/>`rest`<br/>`standalone-grpc`<br/>`standalone-rest`<br/>`jobs/rotatekeys` |

The gRPC server exposes private-key operations and must run on an isolated, access-controlled network — the server does not authenticate callers itself.

<details>
<summary>Optional configuration (REST tuning, OpenTelemetry)</summary>

REST tuning (images `rest`, `standalone-rest`):

| Name                          | Description                          | Default          |
| ----------------------------- | ------------------------------------ | ---------------- |
| `REST_MAX_REQUEST_SIZE`       | Maximum request body size, in bytes. | `2097152` (2MiB) |
| `REST_TIMEOUT_READ`           | Read timeout.                        | `15s`            |
| `REST_TIMEOUT_READ_HEADER`    | Header read timeout.                 | `3s`             |
| `REST_TIMEOUT_WRITE`          | Write timeout.                       | `30s`            |
| `REST_TIMEOUT_IDLE`           | Idle keep-alive timeout.             | `60s`            |
| `REST_TIMEOUT_REQUEST`        | Per-request timeout.                 | `60s`            |
| `REST_CORS_ALLOWED_ORIGINS`   | CORS allowed origins.                | `*`              |
| `REST_CORS_ALLOWED_HEADERS`   | CORS allowed headers.                | `*`              |
| `REST_CORS_ALLOW_CREDENTIALS` | CORS allow-credentials flag.         | `false`          |
| `REST_CORS_MAX_AGE`           | CORS max-age, in seconds.            | `3600`           |

Logs and tracing — OpenTelemetry supports a stdout and a Google Cloud exporter (all server images):

| Name                | Description                                                           | Default             |
| ------------------- | --------------------------------------------------------------------- | ------------------- |
| `OTEL`              | Enable OTel tracing; the variables below pick the exporter.           | `false`             |
| `GCLOUD_PROJECT_ID` | Google Cloud project ID. When set, switches the OTel exporter to GCP. |                     |
| `APP_NAME`          | Application name attached to traces and logs.                         | `service-json-keys` |

</details>

## Using the client packages

Two clients ship with the service. Each snippet is the **minimum viable call**; the full surface is what your editor's intellisense, [pkg.go.dev](https://pkg.go.dev/github.com/a-novel/service-json-keys/v2), and the [API reference](https://a-novel.github.io/service-json-keys-v2) are for.

- **Go** talks gRPC — use it from a backend service that signs tokens or verifies them with cached public keys.
- **JavaScript / TypeScript** talks REST — use it from a frontend or Node service that only needs public keys for local verification.

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

	// In production, swap insecure.NewCredentials() for a TLS or mTLS credential — the
	// server has no application-layer auth, so transport security is the only thing
	// protecting private-key operations from a network adversary.
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
	log.Printf("verified claims for user %s", claims.UserID)
}
```

### JavaScript / TypeScript (REST)

The package is published to GitHub Packages, which requires a Personal Access Token with the `read:packages` scope even for public packages ([why](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193)). Add to `.npmrc` (project root or `$HOME`):

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

```bash
pnpm add @a-novel/service-json-keys-rest
```

```typescript
import { JsonKeysApi, jwkList } from "@a-novel/service-json-keys-rest";

const api = new JsonKeysApi("http://service-json-keys:8080");

// Fetch the active public keys for a usage; cache them client-side and verify locally.
const keys = await jwkList(api, "auth");
```

API reference: [a-novel.github.io/service-json-keys-v2](https://a-novel.github.io/service-json-keys-v2).

## Running locally

For a throwaway instance without the dev toolchain, the **standalone** images bundle the server and migrations in one container. They run migrations on every boot — handy for a quick spin-up, unsafe under multi-replica production restarts.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.3.1
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/standalone-grpc:v2.3.1 # or standalone-rest
    ports: ["${GRPC_PORT}:8080"] # map ${REST_PORT} for the standalone-rest image
    depends_on:
      postgres-json-keys: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks: [api]

networks:
  api:
```

Working on the service itself? Use the `a-novel` CLI (`a-novel run start service-json-keys/grpc`) instead — see [CONTRIBUTING](./CONTRIBUTING.md).

## Contributing

Platform setup and the day-to-day commands live in the [developer onboarding guide](https://github.com/a-novel-kit/.github/blob/master/README.md). Service-specific concepts and local interactions are in [CONTRIBUTING.md](./CONTRIBUTING.md).
