#  Run tests.
test-unit:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

test-pkg:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.pkg.sh'"

test: test-unit test-pkg

# Check code quality.
lint:
	go tool golangci-lint run
	go tool buf lint
	pnpm lint

# Reformat code so it passes the code style lint checks.
format:
	go mod tidy
	go tool golangci-lint run --fix
	go tool buf format -w
	go tool buf dep update
	pnpm format

# Generate Go code.
generate-go:
	go generate ./...

generate: generate-go

# Run the rest API
run:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

build:
	bash -c "set -m; bash '$(CURDIR)/scripts/build.sh'"

install:
	go mod tidy
	pnpm i --frozen-lockfile
