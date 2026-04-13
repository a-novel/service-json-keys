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

## Usage

### Docker

Run the service as a containerized application (the below examples use docker-compose syntax).

#### gRPC

> Set the SERVICE_JSON_KEYS_GRPC_PORT env variable to whatever port you want to use for the service.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.2.6
    networks:
      - api
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
    ports:
      - "${SERVICE_JSON_KEYS_GRPC_PORT}:8080"
    depends_on:
      postgres-json-keys:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks:
      - api

networks:
  api:

volumes:
  json-keys-postgres-data:
```

Note the standalone image is an all-in-one initializer for the application; however, it runs heavy operations such
as migrations on every launch. Thus, while it comes in handy for local development, it is NOT RECOMMENDED for
production deployments. Instead, consider using the separate, optimized images for that purpose.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.2.6
    networks:
      - api
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
      postgres-json-keys:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
    networks:
      - api

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/grpc:v2.2.6
    ports:
      - "${SERVICE_JSON_KEYS_GRPC_PORT}:8080"
    depends_on:
      postgres-json-keys:
        condition: service_healthy
      migrations-json-keys:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks:
      - api

networks:
  api:

volumes:
  json-keys-postgres-data:
```

#### REST

> Set the SERVICE_JSON_KEYS_REST_PORT env variable to whatever port you want to use for the service.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.2.6
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - json-keys-postgres-data:/var/lib/postgresql/

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/standalone-rest:v2.2.6
    ports:
      - "${SERVICE_JSON_KEYS_REST_PORT}:8080"
    depends_on:
      postgres-json-keys:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks:
      - api

networks:
  api:

volumes:
  json-keys-postgres-data:
```

Note the standalone image is an all-in-one initializer for the application; however, it runs heavy operations such
as migrations on every launch. Thus, while it comes in handy for local development, it is NOT RECOMMENDED for
production deployments. Instead, consider using the separate, optimized images for that purpose.

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.2.6
    networks:
      - api
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
      postgres-json-keys:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
    networks:
      - api

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/rest:v2.2.6
    ports:
      - "${SERVICE_JSON_KEYS_REST_PORT}:8080"
    depends_on:
      postgres-json-keys:
        condition: service_healthy
      migrations-json-keys:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
      APP_MASTER_KEY: "<your-master-key-here>"
    networks:
      - api

networks:
  api:

volumes:
  json-keys-postgres-data:
```

Above are the minimal required configuration to run the service locally. Configuration is done through environment
variables. Below is a list of available configurations:

**Required variables**

| Name           | Description                                                                                                                                                                                            | Images                                                                         |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| POSTGRES_DSN   | The Postgres Data Source Name (DSN) used to connect to the database.                                                                                                                                   | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest`<br/>`migrations` |
| APP_MASTER_KEY | Master key used to securely encrypt private keys in the database. This should NEVER be exposed, and should not be rotated unless necessary (changing it will lose access to previous keys permanently) | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest`                  |

The gRPC service exposes sensitive data, and should run in an isolated, secure network.

**REST API**

While you should not need to change these values in most cases, the following variables allow you to
customize the REST API behavior.

| Name                        | Description                                 | Default value    | Images                       |
| --------------------------- | ------------------------------------------- | ---------------- | ---------------------------- |
| REST_MAX_REQUEST_SIZE       | Maximum size of incoming requests in bytes  | `2097152` (2MiB) | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_READ           | Timeout for read operations                 | `15s`            | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_READ_HEADER    | Timeout for header reading operations       | `3s`             | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_WRITE          | Timeout for write operations                | `30s`            | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_IDLE           | Idle timeout                                | `60s`            | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_REQUEST        | Timeout for api requests                    | `60s`            | `standalone-rest`<br/>`rest` |
| REST_CORS_ALLOWED_ORIGINS   | CORS allowed origins (allow all by default) | `*`              | `standalone-rest`<br/>`rest` |
| REST_CORS_ALLOWED_HEADERS   | CORS allowed headers (allow all by default) | `*`              | `standalone-rest`<br/>`rest` |
| REST_CORS_ALLOW_CREDENTIALS | CORS allow credentials                      | `false`          | `standalone-rest`<br/>`rest` |
| REST_CORS_MAX_AGE           | CORS max age                                | `3600`           | `standalone-rest`<br/>`rest` |

**Logs & Tracing**

For now, OTEL is only provided using 2 exporters: stdout and Google Cloud. Other integrations may come
in the future.

| Name              | Description                                                                             | Default value       | Images                                                        |
| ----------------- | --------------------------------------------------------------------------------------- | ------------------- | ------------------------------------------------------------- |
| OTEL              | Activate OTEL tracing (use options below to switch between exporters)                   | `false`             | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |
| GCLOUD_PROJECT_ID | Google Cloud project id for the OTEL exporter. Switch to Google Cloud exporter when set |                     | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |
| APP_NAME          | Application name to be used in traces                                                   | `service-json-keys` | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |

### Javascript (npm)

To interact with a running REST instance of the JSON Keys service, you can use the integrated package.

> ⚠️ **Warning**: Even though the package is public, GitHub registry requires you to have a Personal Access Token
> with `repo` and `read:packages` scopes to pull it in your project. See
> [this issue](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193) for more information.

Make sure you have a `.npmrc` with the following content (in your project or in your home directory):

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

Then, install the package using pnpm:

```bash
# pnpm config set auto-install-peers true
#  Or
# pnpm config set auto-install-peers true --location project
pnpm add @a-novel/service-json-keys-rest
```

To use it, create a `JsonKeysApi` instance. A single instance can be shared across your client.

```typescript
import { JsonKeysApi, jwkGet, jwkList } from "@a-novel/service-json-keys-rest";

export const jsonKeysApi = new JsonKeysApi("<base_api_url>");

// (optional) check the status of the api connection.
await jsonKeysApi.ping();
await jsonKeysApi.health();
```

Retrieve JWK keys:

```typescript
// List all keys for a given usage.
const keys = await jwkList(jsonKeysApi, "auth");

// Retrieve a specific key by ID.
const key = await jwkGet(jsonKeysApi, "<key-id>");
```

The API reference is available at [GitHub Pages](https://a-novel.github.io/service-json-keys-v2).

### Go module

You can integrate the json keys capabilities directly into your Go services by using the provided
Go module. It requires a connection to a running instance of this service.

```bash
go get -u github.com/a-novel/service-json-keys/v2
```

```go
package main

import (
  "context"

  "github.com/a-novel-kit/golib/grpcf"
  "github.com/a-novel/service-json-keys/v2/pkg/go"
)

type MyClaims struct {
  UserID string `json:"userID"`
}

func main() {
  ctx := context.Background()

  client, _ := servicejsonkeys.NewClient(ctx, "<service-json-keys-url>")
  claimsVerifier := servicejsonkeys.NewClaimsVerifier[MyClaims](client)

  claims := MyClaims{ UserID: "user-1" }
  claimsPayload, _ := grpcf.InterfaceToProtoAny(claims)

  // Signed token ready for use.
  token, _ := client.ClaimsSign(ctx, &servicejsonkeys.ClaimsSignRequest{
    Usage: servicejsonkeys.KeyUsageAuth,
    Payload: claimsPayload,
  })

  decodedClaims, _ := claimsVerifier.VerifyClaims(ctx, &servicejsonkeys.VerifyClaimsRequest{
    Usage: servicejsonkeys.KeyUsageAuth,
    AccessToken: token.GetToken(),
  })

  // decodedClaims should be the same as claims.
}
```
