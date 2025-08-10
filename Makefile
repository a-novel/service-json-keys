# Define tool versions.
GCI="github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.1"

# Run tests.
test:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

# Check code quality.
lint:
	go run ${GCI} run
	npx prettier . --check
	sqlfluff lint

# Reformat code so it passes the code style lint checks.
format:
	go mod tidy
	go run ${GCI} run --fix
	npx prettier . --write
	sqlfluff fix

# Lint OpenAPI specs.
openapi-lint:
	npx @redocly/cli lint ./docs/api.yaml

# Generate Go code.
go-generate:
	go generate ./...

run-infra:
	podman compose -p "${APP_NAME}" -f "${PWD}/build/podman-compose.yaml" up -d --build --pull-always

run-infra-down:
	podman compose -p "${APP_NAME}" -f "${PWD}/build/podman-compose.yaml" down

# Execute the keys rotation job locally.
run-rotate-keys:
	bash -c "set -m; bash '$(CURDIR)/scripts/run_rotate_keys.sh'"

# Run the API
run-api:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

install:
	pipx install sqlfluff
	bash -c "cd ./docs && pnpm i"
