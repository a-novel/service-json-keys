# ================================================================================
# Tests
# ================================================================================

# Runs internal Go integration tests (./internal/...).
test-unit:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

# Runs Go client package integration tests (./pkg/...).
test-pkg:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.pkg.sh'"

# Runs TypeScript/JS REST client integration tests.
test-pkg-js:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.pkg.js.sh'"

# Runs the complete test suite (unit, Go pkg, JS pkg). Reserved for pre-commit validation —
# use test-unit or test-pkg during incremental development.
test: test-unit test-pkg test-pkg-js

# ================================================================================
# Lint
# ================================================================================

# Checks Go source for style and correctness issues. Run before committing or in CI.
lint-go:
	go tool -modfile=golangci-lint.mod golangci-lint run ./...

# Validates protobuf definitions against buf's lint rules. Run after editing .proto files.
lint-proto:
	go tool buf lint

# Validates TypeScript/JS source and the OpenAPI spec. Run before committing JS changes.
lint-node:
	pnpm lint

lint: lint-go lint-proto lint-node

# ================================================================================
# Format
# ================================================================================

# Also runs go mod tidy and applies auto-fixable golangci-lint corrections.
format-go:
	go mod tidy
	go tool -modfile=golangci-lint.mod golangci-lint run ./... --fix

# Formats .proto files and syncs buf.lock. Run both steps together after editing
# protobuf definitions — buf.lock must stay consistent with the formatted output.
format-proto:
	go tool buf format -w
	go tool buf dep update

# Formats TypeScript/JS source and the OpenAPI spec with Prettier.
format-node:
	pnpm format

format: format-go format-proto format-node

# ================================================================================
# Generate
# ================================================================================

# Regenerates derived Go code. Run after editing .proto files or any interface with mocks.
generate-go:
	go generate ./...

generate: generate-go

# ================================================================================
# Local dev
# ================================================================================
run:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

build:
	bash -c "set -m; bash '$(CURDIR)/scripts/build.sh'"

# Installs all Go and Node dependencies. Run after cloning or after adding new dependencies.
# --frozen-lockfile ensures Node packages match pnpm-lock.yaml exactly; use pnpm update to
# intentionally upgrade Node dependencies.
install:
	go mod tidy
	pnpm i --frozen-lockfile
