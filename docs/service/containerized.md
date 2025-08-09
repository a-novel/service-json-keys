---
outline: deep
---

# Containerized

You can import the JSON keys service as a container. You need to import both API and associated Jobs for the
service to run correctly.

::: code-group

```yaml [podman]
# https://github.com/containers/podman-compose
services:
  json-keys-postgres:
    image: ghcr.io/a-novel/service-json-keys/database:v1
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: json-keys
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - json-keys-postgres-data:/var/lib/postgresql/data/

  json-keys-postgres-migrations:
    image: ghcr.io/a-novel/service-json-keys/jobs/migrations:v1
    depends_on:
      json-keys-postgres:
        condition: service_healthy
    networks:
      - api
    environment:
      POSTGRES_DSN: postgres://postgres:postgres@json-keys-postgres:5432/json-keys?sslmode=disable

  # Make sure the master key is the same across all containers.
  # The Master Key is a secure, 32-bit random secret used to encrypt private JSON keys
  # in the database.
  # This value is a dummy key used for tests. Use your own random key in production.

  json-keys-job-rotate-keys:
    image: ghcr.io/a-novel/service-json-keys/jobs/rotatekeys:v1
    depends_on:
      json-keys-postgres:
        condition: service_healthy
      json-keys-postgres-migrations:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: postgres://postgres:postgres@json-keys-postgres:5432/json-keys?sslmode=disable
      APP_MASTER_KEY: fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca
    networks:
      - api

  json-keys-service:
    image: ghcr.io/a-novel/service-json-keys/api:v1
    depends_on:
      json-keys-postgres:
        condition: service_healthy
      json-keys-postgres-migrations:
        condition: service_completed_successfully
      json-keys-job-rotate-keys:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: postgres://postgres:postgres@json-keys-postgres:5432/json-keys?sslmode=disable
      APP_MASTER_KEY: fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca
    networks:
      - api

networks:
  api: {}

volumes:
  json-keys-postgres-data:
```

:::

## Standalone image (local)

For local development or CI purposes, you can also load a standalone version that runs all the necessary jobs
before starting the service.

::: warning
The standalone image takes longer to boot, and it is not suited for production use.
:::

::: code-group

```yaml [podman]
# https://github.com/containers/podman-compose
services:
  json-keys-postgres:
    image: ghcr.io/a-novel/service-json-keys/database:v1
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: json-keys
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - json-keys-postgres-data:/var/lib/postgresql/data/

  # The Master Key is a secure, 32-bit random secret used to encrypt private JSON keys
  # in the database.
  # This value is a dummy key used for tests. Use your own random key in production.
  json-keys-service:
    image: ghcr.io/a-novel/service-json-keys/standalone:v1
    depends_on:
      json-keys-postgres:
        condition: service_healthy
    environment:
      POSTGRES_DSN: postgres://postgres:postgres@json-keys-postgres:5432/json-keys?sslmode=disable
      APP_MASTER_KEY: fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca
    networks:
      - api

networks:
  api: {}

volumes:
  json-keys-postgres-data:
```

:::

## Configuration

Configuration is done through environment variables.

### Required variables

You must provide the following variables for the service to run correctly.

| Variable         | Description                                                                                         | Images                                 |
| ---------------- | --------------------------------------------------------------------------------------------------- | -------------------------------------- |
| `APP_MASTER_KEY` | The Master Key is a secure, 32-bit random secret used to encrypt private JSON keys in the database. | `standalone`, `api`, `jobs/rotatekeys` |
| `POSTGRES_DSN`   | Connection string to the Postgres database.                                                         | `standalone`, `api`, `jobs/rotatekeys` |

### Optional variables

Generic configuration.

| Variable   | Description                                     | Default                                                     | Images                                 |
| ---------- | ----------------------------------------------- | ----------------------------------------------------------- | -------------------------------------- |
| `APP_NAME` | Name of the application, used for tracing.      | `json-keys-service`<br/>`service-json-keys-job-rotate-keys` | `standalone`, `api`, `jobs/rotatekeys` |
| `ENV`      | Provide information on the current environment. |                                                             | `standalone`, `api`, `jobs/rotatekeys` |

API configuration.

| Variable                     | Description                                                                             | Default | Images              |
| ---------------------------- | --------------------------------------------------------------------------------------- | ------- | ------------------- |
| `API_PORT`                   | Port to run the API on.                                                                 | `8080`  | `standalone`, `api` |
| `API_MAX_REQUEST_SIZE`       | Maximum request size for the API.<br/>Provided as a number of bytes.                    | `2MB`   | `standalone`, `api` |
| `API_TIMEOUT_READ`           | Read timeout for the API.<br/>Provided as a duration string.                            | `5s`    | `standalone`, `api` |
| `API_TIMEOUT_READ_HEADER`    | Header read timeout for the API.<br/>Provided as a duration string.                     | `3s`    | `standalone`, `api` |
| `API_TIMEOUT_WRITE`          | Write timeout for the API.<br/>Provided as a duration string.                           | `10s`   | `standalone`, `api` |
| `API_TIMEOUT_IDLE`           | Idle timeout for the API.<br/>Provided as a duration string.                            | `30s`   | `standalone`, `api` |
| `APITimeoutRequest`          | Request timeout for the API.<br/>Provided as a duration string.                         | `15s`   | `standalone`, `api` |
| `API_CORS_ALLOWED_ORIGINS`   | CORS allowed origins for the API.<br/>Provided as a list of values separated by commas. | `*`     | `standalone`, `api` |
| `API_CORS_ALLOWED_HEADERS`   | CORS allowed headers for the API.<br/>Provided as a list of values separated by commas. | `*`     | `standalone`, `api` |
| `API_CORS_ALLOW_CREDENTIALS` | Whether to allow credentials in CORS requests.                                          | `false` | `standalone`, `api` |
| `API_CORS_MAX_AGE`           | CORS max age for the API.<br/>Provided as a number of seconds.                          | `3600`  | `standalone`, `api` |

Tracing configuration.

| Variable            | Description                                                                                                                      | Default |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `GCLOUD_PROJECT_ID` | A Google Cloud Platform project ID. Note this activate Google Cloud tracing.                                                     |         |
| `PRETTY_CONSOLE`    | When using local (default) tracing, set it to false to turn off colors in the console. Ineffective if `GCLOUD_PROJECT_ID` is set | `true`  |
