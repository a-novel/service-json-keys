# JSON Keys service

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

<hr />

This is a quickstart document to test the project locally.

You can find the API documentation on the [repository GitHub page](https://a-novel.github.io/service-json-keys/).

Want to contribute? Check the [contribution guidelines](CONTRIBUTING.md).

# Use in a project

You can import this application as a docker image. Below is an example using
[podman compose](https://docs.podman.io/en/latest/markdown/podman-compose.1.html).

```yaml
services:
  json-keys-postgres:
    image: docker.io/library/postgres:17
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

  # Runs the secret key rotation on every launch.
  # Keys are smartly rotated, meaning new keys are generated only when necessary
  # (eg: keys missing or last generated version is too old).
  # The container will exit by itself when the job is done.
  json-keys-rotate-keys-job:
    image: ghcr.io/a-novel/service-json-keys/jobs/rotatekeys:v0
    depends_on:
      - json-keys-postgres
    environment:
      ENV: local
      APP_NAME: json-keys-service-rotate-keys-job
      DSN: postgres://postgres:postgres@json-keys-postgres:5432/json-keys?sslmode=disable
      # Dummy key used only for local environment. Consider using a secure, private key in production.
      # Note it MUST match the one used in the json-keys service.
      MASTER_KEY: fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca
      # Used for tracing purposes, can be omitted.
      # SENTRY_DSN: [your_sentry_dsn]
      # SERVER_NAME: json-keys-service-prod
      # RELEASE: v0.1.2
      # ENV: production
      # Set the following if you want to debug the service locally.
      # DEBUG: true
    networks:
      - api

  json-keys-service:
    image: ghcr.io/a-novel/service-json-keys/api:v0
    depends_on:
      - json-keys-postgres
    ports:
      # Expose the service on port 4001 on the local machine.
      - "4021:8080"
    environment:
      PORT: 8080
      ENV: local
      APP_NAME: json-keys-service
      DSN: postgres://postgres:postgres@json-keys-postgres:5432/json-keys?sslmode=disable
      # Dummy key used only for local environment. Consider using a secure, private key in production.
      # Note it MUST match the one used in the json-keys keys rotation job.
      MASTER_KEY: fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca
      AUTH_PLATFORM_URL_UPDATE_EMAIL: http://localhost:4001/update-email
      AUTH_PLATFORM_URL_UPDATE_PASSWORD: http://localhost:4001/update-password
      AUTH_PLATFORM_URL_REGISTER: http://localhost:4001/register
      # Used for tracing purposes, can be omitted.
      # SENTRY_DSN: [your_sentry_dsn]
      # SERVER_NAME: json-keys-service-prod
      # RELEASE: v0.1.2
      # ENV: production
      # Set the following if you want to debug the service locally.
      # DEBUG: true
    networks:
      - api

networks:
  api: {}

volumes:
  json-keys-postgres-data:
```

Available tags includes:

- `latest`: latest versioned image
- `vx`: versioned images, pointing to a specific version. Partial versions are supported. When provided, the
  latest subversion is used.\
  examples: `v0`, `v0.1`, `v0.1.2`
- `branch`: get the latest version pushed to a branch. Any valid branch name can be used.\
  examples: `master`, `fix/something`

# Run locally

## Pre-requisites

- [Golang](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download/)
- [Python](https://www.python.org/downloads/)
  - Install [pipx](https://pipx.pypa.io/stable/installation/) to install command-line tools.
- [Podman](https://podman.io/docs/installation)
  - Install [podman-compose](https://github.com/containers/podman-compose)

  ```bash
  # Pipx
  pipx install podman-compose

  # Brew
  brew install podman-compose
  ```

- Make

  ```bash
  # Debian / Ubuntu
  sudo apt-get install build-essential

  # macOS
  brew install make
  ```

  For Windows, you can use [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

## Setup environment

Create a `.envrc` file from the template:

```bash
cp .envrc.template .envrc
```

Then fill the missing secret variables. Once your file is ready:

```bash
source .envrc
```

> You may use tools such as [direnv](https://direnv.net/), otherwise you'll need to source the env file on each new
> terminal session.

Install the external dependencies:

```bash
make install
```

## Run infrastructure

```bash
make run-infra
# To turn down:
# make run-infra-down
```

> You may skip this step if you already have the global infrastructure running.

## Generate keys

You need to do this at least once, to have a set of keys ready to use for json-keys.

> It is recommended to run this regularly, otherwise keys will expire and json-keys
> will fail.

```bash
make run-rotate-keys

# [Sentry] 2025/06/26 14:00:59 generated new key for usage auth: e70eaf3f-1861-4be7-80c2-85c34e9b8371
# [Sentry] 2025/06/26 14:00:59 generated new key for usage refresh: cd4be805-6fed-4b50-8d6a-3e1fcd65e3c8
```

## Et Voilà!

```bash
make run-api
```

# Use in other services

The exported methods are available under the `/pkg` component.

```bash
go get github.com/a-novel/service-json-keys
```

## Setup

You need an API client to interact with the running service. Make sure to provide the correct URL:

```go
package main

import jkPkg "github.com/a-novel/service-json-keys/pkg"

client, err := jkPkg.NewClient(serverURL)
```

## Sign / Verify claims

You can produce and consume JWTs. To do so, you need to select a config. Available configs are listed using
the `models.KeyUsage` constants.

```go
package main

import jkModels "github.com/a-novel/service-json-keys/models"

// Tokens signed with a given usage are only valid for that usage.
usage := jkModels.KeyUsageAuth
```

Sign a claim

```go
package main

import jkPkg "github.com/a-novel/service-json-keys/pkg"

type CustomClaims struct {
	UserID string `json:"userID"`
}

// You can share a single signer for all usages.
signer := jkPkg.NewClaimsSigner(client)

token, err := signer.SignClaims(ctx, usage, &CustomClaims{)
	UserID: "1234",
})
```

Consume claims

```go
package main

import jkPkg "github.com/a-novel/service-json-keys/pkg"

type CustomClaims struct {
	UserID string `json:"userID"`
}

// Each claim verifier is only targeted at a specific claims body type.
// You may share a single verifier for multiple usages, if the token
// body type is the same.
verifier, err := jkPkg.NewClaimsVerifier[CustomClaims](client)

// Note the token is also validated, meaning expired tokens will return an error.
claims, err := signer.SignClaims(ctx, usage, token, nil)
```
