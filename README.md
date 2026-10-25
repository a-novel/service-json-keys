# JSON Keys service

Holds the signing keys of A-Novel services, signs their tokens and serves the public keys that verify them.

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-json-keys)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-json-keys)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-json-keys)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-json-keys/main.yaml)
[![codecov](https://codecov.io/gh/a-novel/service-json-keys/graph/badge.svg)](https://codecov.io/gh/a-novel/service-json-keys)

![Coverage graph](https://codecov.io/gh/a-novel/service-json-keys/graphs/sunburst.svg)

## What it does

Services register named **usages**, such as `auth` or `auth-refresh`. Each usage has its own signing algorithm, rotation schedule and token claims. JSON Keys holds every private key and signs on behalf of its callers, so key material never leaves the server. Consumers cache the public keys and verify tokens locally, with no call per token.

Other services sign tokens and read keys through the private gRPC API. The gRPC server authenticates no caller, so network policy, an ingress or a service mesh decides who reaches it. The public REST API serves the public keys and a health check to anyone verifying tokens.

## Deploying

The service runs as published OCI images plus a PostgreSQL database. All state lives in the database, so each server scales to any number of replicas behind a load balancer.

Deploy the images with any container orchestrator. The composition below shows which images to run, how they wire together, and the environment they expect.

| Image                               | Role                                                                             |
| ----------------------------------- | -------------------------------------------------------------------------------- |
| `service-json-keys/grpc`            | Private signing + key-management API. Internal network only.                     |
| `service-json-keys/rest`            | Public key-fetch + health API.                                                   |
| `service-json-keys/jobs/migrations` | One-shot schema migration job; runs to completion before the servers start.      |
| `service-json-keys/jobs/rotatekeys` | Scheduled key-rotation job (see [CONTRIBUTING](./CONTRIBUTING.md#key-rotation)). |
| `service-json-keys/database`        | PostgreSQL with pgBackRest, on Wolfi. Any PostgreSQL server works too.           |

Pin every image to the same tag, the [latest release](https://github.com/a-novel/service-json-keys/releases/latest) for a new deployment. Start `database`, run the migrations job to completion, then start any number of `grpc` and `rest` replicas.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.13.2
    networks: [api]
    environment:
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_DB: "${POSTGRES_DATABASE}"
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - json-keys-postgres-data:/var/lib/postgresql/
    # Podman ignores the image's own check, so the dependents' wait needs this one.
    healthcheck:
      test: ["CMD", "pg_isready"]
      interval: 1s
      timeout: 5s
      retries: 10

  migrations-json-keys:
    image: ghcr.io/a-novel/service-json-keys/jobs/migrations:v2.13.2
    depends_on:
      postgres-json-keys: { condition: service_healthy }
    environment:
      POSTGRES_HOST: postgres-json-keys
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
    networks: [api]

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/grpc:v2.13.2 # or .../rest for the public REST API
    ports: ["${GRPC_PORT}:8080"] # the container always listens on 8080; map ${REST_PORT} for the rest image
    depends_on:
      postgres-json-keys: { condition: service_healthy }
      migrations-json-keys: { condition: service_completed_successfully }
    environment:
      POSTGRES_HOST: postgres-json-keys
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks: [api]

networks:
  api:

volumes:
  json-keys-postgres-data:
```

To serve REST as well, add a service with the `rest` image, the same database and no `APP_MASTER_KEY`.

Schedule the `jobs/rotatekeys` image, and run it once before the servers take traffic. Keys expire after their TTL and only this job replaces them, so signing stops without it.

### Database image

The database image runs PostgreSQL and pgBackRest on Wolfi. The container build assembles its
[package manifest](./builds/database.apko.yaml) with apko, so Docker and Podman need no extra
host tool. PostgreSQL keeps its standard `POSTGRES_*` initialization variables, the `uuid-ossp`
extension and its volume at `/var/lib/postgresql`. You schedule the backups and hold the
repository credentials.

A database moving from the Debian-based image needs a fresh volume. Its data directory and
physical backups do not carry over, because the libc, collation and extension environment
changed. To keep the data, restore a logical dump into the new database and validate it before
switching traffic. Keep the previous image with its backups.

Wolfi's package repository rolls forward, so an old package manifest may stop building. Keep the
published service images for recovery.

### Configuration

| Name                   | Description                                                                                                                                                                                                                              | Images                                                                   |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `POSTGRES_HOST`        | PostgreSQL hostname or IP address. **Required.**                                                                                                                                                                                         | all                                                                      |
| `POSTGRES_PORT`        | PostgreSQL port. Defaults to `5432`.                                                                                                                                                                                                     | all                                                                      |
| `POSTGRES_USER`        | PostgreSQL login role. **Required.**                                                                                                                                                                                                     | all                                                                      |
| `POSTGRES_PASSWORD`    | PostgreSQL login password. **Required; inject it as a secret.**                                                                                                                                                                          | all                                                                      |
| `POSTGRES_DATABASE`    | PostgreSQL database name. **Required.**                                                                                                                                                                                                  | all                                                                      |
| `POSTGRES_TLS_ENABLED` | Encrypt the PostgreSQL connection. Defaults to `true`; disable only when another trusted boundary protects the database link.                                                                                                            | all                                                                      |
| `APP_MASTER_KEY`       | Hex-encoded 32-byte key that encrypts private keys at rest. **Required** by every image that signs or rotates keys. Changing it leaves every stored private key unreadable: see [CONTRIBUTING](./CONTRIBUTING.md#master-key-encryption). | `grpc`<br/>`standalone-grpc`<br/>`standalone-rest`<br/>`jobs/rotatekeys` |

The `rest` image never decrypts a private key, so deploy it without the master key or any right to read it. It still reads the shared key table with database credentials, so it gives no database isolation.

<details>
<summary>Optional configuration (server tuning, OpenTelemetry)</summary>

gRPC tuning (images `grpc`, `standalone-grpc`):

| Name                    | Description               | Default |
| ----------------------- | ------------------------- | ------- |
| `GRPC_TIMEOUT_SHUTDOWN` | Graceful shutdown budget. | `30s`   |

REST tuning (images `rest`, `standalone-rest`):

| Name                          | Description                                                                                             | Default         |
| ----------------------------- | ------------------------------------------------------------------------------------------------------- | --------------- |
| `REST_MAX_REQUEST_SIZE`       | Maximum request body size, in bytes.                                                                    | `16384` (16KiB) |
| `REST_CLIENT_IP_HEADER`       | Header a trusted proxy sets to the visitor IP. Set it only when every request comes through that proxy. | TCP peer        |
| `REST_LIVENESS_PATH`          | Ping path, which still answers during a planned downtime and skips the rate limit.                      | `/v2/ping`      |
| `REST_RATE_LIMITS_ENABLED`    | Rate-limit each visitor on every route but the ping. Turn off only in test stacks.                      | `true`          |
| `REST_RATE_LIMIT`             | Requests each visitor sends to each route, as `<requests>/<window>`.                                    | `5/1s`          |
| `REST_TIMEOUT_READ`           | Read timeout.                                                                                           | `15s`           |
| `REST_TIMEOUT_READ_HEADER`    | Header read timeout.                                                                                    | `3s`            |
| `REST_TIMEOUT_WRITE`          | Write timeout.                                                                                          | `30s`           |
| `REST_TIMEOUT_IDLE`           | Idle keep-alive timeout.                                                                                | `60s`           |
| `REST_TIMEOUT_REQUEST`        | Per-request timeout.                                                                                    | `60s`           |
| `REST_TIMEOUT_SHUTDOWN`       | Graceful shutdown budget.                                                                               | `30s`           |
| `REST_CORS_ALLOWED_ORIGINS`   | CORS allowed origins.                                                                                   | `*`             |
| `REST_CORS_ALLOWED_HEADERS`   | CORS allowed headers.                                                                                   | `*`             |
| `REST_CORS_ALLOWED_METHODS`   | CORS allowed methods, comma-separated.                                                                  | `HEAD,GET`      |
| `REST_CORS_ALLOW_CREDENTIALS` | CORS allow-credentials flag.                                                                            | `false`         |
| `REST_CORS_MAX_AGE`           | CORS max-age, in seconds.                                                                               | `3600`          |

Database connection pool (server images). Each process holds its own pool, so the database's `max_connections`, 100 by default, must cover every replica and the migration job.

| Name                      | Description                               | Default |
| ------------------------- | ----------------------------------------- | ------- |
| `POSTGRES_MAX_OPEN_CONNS` | Maximum open connections to the database. | `20`    |
| `POSTGRES_MAX_IDLE_CONNS` | Maximum connections kept open while idle. | `20`    |

Planned downtime (all server images, `jobs/rotatekeys`):

| Name             | Description                                                                                                                                                                                                     | Default |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `DOWNTIME_START` | When a planned downtime starts, in RFC 3339. Until the variable is removed, the servers answer `503` / `UNAVAILABLE` except to ping and gRPC health, and start without the database. Key rotation does nothing. |         |

Logs and tracing (server images, `jobs/rotatekeys`):

| Name                | Description                                                                                                  | Default             |
| ------------------- | ------------------------------------------------------------------------------------------------------------ | ------------------- |
| `OTEL`              | Enable OpenTelemetry tracing.                                                                                | `false`             |
| `GCLOUD_PROJECT_ID` | Google Cloud project ID. When set, logs and traces go to Google Cloud. Otherwise they go to standard output. |                     |
| `APP_NAME`          | Application name attached to traces and logs.                                                                | `service-json-keys` |

</details>

## Using the client packages

- **Go** calls the gRPC API. Use it from a backend service that signs tokens or verifies them.
- **JavaScript / TypeScript** calls the REST API. Use it from a frontend or a Node service that only needs the public keys.

The examples make the minimum call. [pkg.go.dev](https://pkg.go.dev/github.com/a-novel/service-json-keys/v2) and the [API reference](https://a-novel.github.io/service-json-keys-v2) document the full surface.

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

	// Use TLS or mTLS credentials in production. The server authenticates no caller, so
	// transport security alone protects private-key operations.
	client, err := servicejsonkeys.NewClient(
		"service-json-keys:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Sign claims under the auth usage.
	payload, err := grpcf.MarshalJSONAsAny(MyClaims{UserID: "user-1"})
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

	// Verify locally, with no network call per token.
	verifier, err := servicejsonkeys.NewClaimsVerifier[MyClaims](client)
	if err != nil {
		log.Fatal(err)
	}
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

GitHub Packages hosts the package. It needs a personal access token with the `read:packages` scope, even for a public package ([why](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193)). Add the registry to the `.npmrc` of your project or home directory:

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

// Fetch the active public keys of a usage. Cache them and verify tokens locally.
const keys = await jwkList(api, "auth");
```

## Running locally

The **standalone** images run a throwaway instance without the development toolchain. Each bundles a server with the migrations and rotation jobs, and runs both jobs at every boot. Replicas restarting together run them concurrently, so keep standalone images out of production.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.13.2
    networks: [api]
    environment:
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_DB: "${POSTGRES_DATABASE}"
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    # Podman ignores the image's own check, so the dependents' wait needs this one.
    healthcheck:
      test: ["CMD", "pg_isready"]
      interval: 1s
      timeout: 5s
      retries: 10

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/standalone-grpc:v2.13.2 # or standalone-rest
    ports: ["${GRPC_PORT}:8080"] # map ${REST_PORT} for the standalone-rest image
    depends_on:
      postgres-json-keys: { condition: service_healthy }
    environment:
      POSTGRES_HOST: postgres-json-keys
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks: [api]

networks:
  api:
```

To work on the service itself, run it with the `a-novel` CLI instead, as [CONTRIBUTING](./CONTRIBUTING.md) shows.

## Contributing

The [developer guide](https://a-novel-kit.github.io/stack/) sets up the platform. [CONTRIBUTING.md](./CONTRIBUTING.md) covers what is specific to this service.
