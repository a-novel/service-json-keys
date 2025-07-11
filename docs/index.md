---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Service JSON Keys"
  tagline: "Manages JSON web keys and tokens."

features:
  - title: As a service
    details: Import this service in your project, as a standalone container or a Go embedded service.
  - title: As a module
    details: Interact with the service using the provided Go module, shipped with all the definitions and types.
  - title: Developers
    details: Participate in the project development, or create your own forks.
---

<br/>

# Local development

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
