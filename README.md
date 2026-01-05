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

## Import in other projects

## Usage

### Docker

Run the service as a containerized application (the below examples use docker-compose syntax).

```yaml
services:
  postgres-json-keys:
    image: ghcr.io/a-novel/service-json-keys/database:v2.1.2
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
    image: ghcr.io/a-novel/service-json-keys/standalone:v2.1.2
    ports:
      - "4001:8080"
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
    image: ghcr.io/a-novel/service-json-keys/database:v2.1.2
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
    image: ghcr.io/a-novel/service-json-keys/migrations:v2.1.2
    depends_on:
      postgres-json-keys:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-json-keys:5432/postgres?sslmode=disable"
    networks:
      - api

  service-json-keys:
    image: ghcr.io/a-novel/service-json-keys/grpc:v2.1.2
    ports:
      - "4002:8080"
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

| Name           | Description                                                                                                                                                                                            | Images                                   |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------- |
| POSTGRES_DSN   | The Postgres Data Source Name (DSN) used to connect to the database.                                                                                                                                   | `standalone`<br/>`grpc`<br/>`migrations` |
| APP_MASTER_KEY | Master key used to securely encrypt private keys in the database. This should NEVER be exposed, and should not be rotated unless necessary (changing it will lose access to previous keys permanently) | `standalone`<br/>`grpc`                  |

The GRPC service exposes sensitive data, and should run in an isolated, secure network.

**Logs & Tracing**

For now, OTEL is only provided using 2 exporters: stdout and Google Cloud. Other integrations may come
in the future.

| Name              | Description                                                                             | Default value       | Images                  |
| ----------------- | --------------------------------------------------------------------------------------- | ------------------- | ----------------------- |
| OTEL              | Activate OTEL tracing (use options below to switch between exporters)                   | `false`             | `standalone`<br/>`rest` |
| GCLOUD_PROJECT_ID | Google Cloud project id for the OTEL exporter. Switch to Google Cloud exporter when set |                     | `standalone`<br/>`rest` |
| APP_NAME          | Application name to be used in traces                                                   | `service-json-keys` | `standalone`<br/>`rest` |

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
  jkpkg "github.com/a-novel/service-json-keys/v2/pkg"
)

type MyClaims struct {
  UserID string `json:"userID"`
}

func main() {
  ctx := context.Background()

  client, _ := jkpkg.NewClient(ctx, "<service-json-keys-url>")
  claimsVerifier := jkpkg.NewClaimsVerifier[MyClaims](client)

  claims := MyClaims{ UserID: "user-1" }
  claimsPayload, _ := grpcf.InterfaceToProtoAny(claims)

  // Signed token ready for use.
  token, _ := client.ClaimsSign(ctx, &jkpkg.ClaimsSignRequest{
    Usage: jkpkg.KeyUsageAuth,
    Payload: claimsPayload,
  })

  decodedClaims, _ := claimsVerifier.VerifyClaims(ctx, &jkpkg.VerifyClaimsRequest{
    Usage: jkpkg.KeyUsageAuth,
    AccessToken: token.GetToken(),
  })

  // decodedClaims should be the same as claims.
}
```
