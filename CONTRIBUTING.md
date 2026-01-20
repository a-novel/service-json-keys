# Contributing to service-json-keys

Welcome to the JSON Keys service for the A-Novel platform. This guide will help you understand the codebase, set
up your development environment, and contribute effectively.

## Table of Contents

1. [Introduction](#introduction)
2. [Quick Start](#quick-start)
3. [Project Architecture](#project-architecture)
4. [Development Workflow](#development-workflow)
5. [AI Usage](#ai-usage)
6. [Go Coding Conventions](#go-coding-conventions)
7. [Protocol Buffers](#protocol-buffers)
8. [Testing](#testing)
9. [CI/CD Pipeline](#cicd-pipeline)
10. [Project-Specific Guidelines](#project-specific-guidelines)
11. [Appendix](#appendix)

---

## Introduction

This is a **Go backend service** providing JWK (JSON Web Key) management for the A-Novel platform:

- JWK storage with encryption (private keys secured via master key)
- JWT signing and verification
- Key rotation and lifecycle management
- gRPC API for internal service communication

The project exposes a **Go client package** (`pkg/`) for other services to integrate with. There is no JavaScript/TypeScript client — this service is designed for internal backend-to-backend communication only.

**Tech Stack:**

| Layer         | Technology             |
| ------------- | ---------------------- |
| Backend       | Go, gRPC               |
| Database      | PostgreSQL             |
| Key Storage   | AES-GCM encrypted JWKs |
| Observability | OpenTelemetry          |
| Containers    | Docker/Podman          |
| Protobuf      | Buf                    |

---

## Quick Start

### Prerequisites

The following must be installed on your system.

- [Go](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download)
  - [pnpm](https://pnpm.io/installation)
- [Podman](https://podman.io/docs/installation)
- (optional) [Direnv](https://direnv.net/)
- Make
  - `sudo apt-get install build-essential` (apt)
  - `sudo pacman -S make` (arch)
  - `brew install make` (macOS)
  - [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

### Bootstrap

Create a `.envrc` file in the project root:

```bash
cp .envrc.template .envrc
```

Then, load the environment variables:

```bash
direnv allow .
# Alternatively, if you don't have direnv on your system
source .envrc
```

Finally, install the dependencies:

```bash
make install
```

### Common Commands

| Command         | Description                      |
| --------------- | -------------------------------- |
| `make run`      | Start all services locally       |
| `make test`     | Run all tests                    |
| `make lint`     | Run all linters                  |
| `make format`   | Format all code                  |
| `make build`    | Build Docker images locally      |
| `make generate` | Generate mocks and protobuf code |

### Interacting with the Service

Once the service is running (`make run`), you can interact with it using `grpcurl` or any gRPC client.

```bash
# List all available methods.
grpcurl -plaintext localhost:4002 list
```

#### Health Checks

```bash
# Simple ping (is the server up?)
grpcurl -plaintext localhost:4002 grpc.health.v1.Health/Check

# Check the status of all services.
grpcurl -plaintext localhost:4002 StatusService/Status
```

#### Key Operations

List available keys:

```bash
grpcurl -plaintext -d '{"usage": "auth"}' localhost:4002 JwkListService/JwkList
```

Get a specific key:

```bash
grpcurl -plaintext -d '{"id": "<key-uuid>"}' localhost:4002 JwkGetService/JwkGet
```

---

## Project Architecture

### Directory Structure

```
service-json-keys-v2/
├── cmd/                          # Entry points (Go)
│   ├── grpc/main.go              # gRPC API server
│   ├── migrations/main.go        # Database migration runner
│   └── rotate-keys/main.go       # Key rotation job
│
├── internal/                     # Private Go packages (core logic)
│   ├── config/                   # Configuration management
│   ├── dao/                      # Data Access Objects (external data sources)
│   ├── handlers/                 # gRPC request handlers
│   │   └── protogen/             # Generated protobuf Go code
│   ├── services/                 # Business logic layer
│   ├── lib/                      # Utilities (master key, encryption)
│   └── models/                   # Domain models
│       ├── migrations/           # SQL migration files
│       └── proto/                # Protocol buffer definitions
│
├── pkg/                          # Public Go packages
│   ├── client.go                 # gRPC client for other services
│   ├── claims.go                 # Claims verification utilities
│   └── jwkExportGrpc.go          # JWK export adapter
│
├── builds/                       # Docker build files
│   ├── *.Dockerfile              # Image definitions
│   └── podman-compose.yaml       # Local development stack
│
├── scripts/                      # Build and test scripts
└── .github/workflows/            # CI/CD pipelines
```

### Layered Architecture

The backend follows a clean layered architecture:

```
┌─────────────────────────────────────────────────────┐
│                     gRPC Layer                      │
│                     (handlers/)                     │
│  • Request parsing, validation, response formatting │
└──────────────────────────┬──────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────┐
│                    Service Layer                    │
│                     (services/)                     │
│  • Business logic, orchestration                    │
│  • Transaction management                           │
│  • Input validation, error handling                 │
└──────────────────────────┬──────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────┐
│                  Data Access Layer                  │
│                       (dao/)                        │
│  • External data source abstraction                 │
│  • Query execution and response mapping             │
└──────────────────────────┬──────────────────────────┘
                           │
                           ▼
                      PostgreSQL
```

### Request Flow

```
gRPC Request
    │
    ▼
grpc.Server (routing + interceptors)
    │
    ▼
Handler.Method()
    ├── Parse protobuf request
    ├── Call service.Exec()
    └── Return protobuf response
           │
           ▼
      Service.Exec()
           ├── Validate request
           ├── Execute business logic
           └── Return result
                  │
                  ▼
             DAO.Exec()
                  ├── Query PostgreSQL
                  └── Map response to domain model
```

---

## Development Workflow

### Git Conventions

#### General Rules

- **Always lowercase** for branch names and commit messages
- `master` is the main branch — create feature branches from it
- One branch per feature/fix — keep changes focused

#### Branch Naming

Format: `type/scope/short-description`

```
feat/services/key-rotation
fix/dao/jwk-select-null-check
chore/deps/upgrade-grpc
refactor/handlers/claims-sign
```

**Types:**

| Type    | Usage                                       |
| ------- | ------------------------------------------- |
| `feat`  | New feature or capability                   |
| `fix`   | Bug fix                                     |
| `chore` | Maintenance (dependencies, config, tooling) |
| `test`  | Adding or updating tests                    |
| `perf`  | Performance improvements                    |

**Scopes** describe the affected area:

- **Layer**: `dao`, `services`, `handlers`, `cmd`, `config`
- **Domain**: `jwk`, `claims`, `encryption`
- **General**: `deps`, `format`, `tests`, `ci`

#### Commit Messages

Format: `type(scope): short description`

```
feat(handlers): add key expiration check
fix(dao): handle null payload in jwk lookup
chore(deps): upgrade testify to v1.9.0
refactor(services): extract key validation logic
docs(readme): update installation instructions
test(claims): add edge cases for token verification
perf(dao): optimize jwk search query
```

Keep descriptions concise and imperative ("add", "fix", "update" — not "added", "fixes", "updating").

#### Hotfixes

`hotfix` is **reserved for direct commits to `master`** in urgent situations. Do not use it for branch names.

```
hotfix(encryption): patch key derivation vulnerability
```

Hotfixes bypass the normal PR flow and should be used sparingly for critical production issues only.

### Making Changes

1. **Create a branch** for your feature or fix
2. **Write tests first** (TDD encouraged)
3. **Implement the change**
4. **Run tests**: `make test`
5. **Run linters**: `make lint`
6. **Format code**: `make format`
7. **Create a pull request**

### Code Generation

The project uses code generation for:

- **Mocks**: Generated via `mockery` from interfaces
- **Protobuf**: Generated via `buf` from `.proto` files

Always run `make generate` after:

- Adding or modifying interfaces that need mocks
- Changing proto files in `internal/models/proto/`

### Database Migrations

Migrations live in `internal/models/migrations/` as embedded SQL files.

To add a new migration:

1. Create a new `.sql` file with a timestamped name (e.g., `20250120150000_add_index.up.sql`)
2. Create the corresponding down migration (`20250120150000_add_index.down.sql`)
3. Write your DDL statements
4. The migration runs automatically on service startup

---

## AI Usage

Using AI assistants (ChatGPT, Claude, Copilot, etc.) is allowed. They can significantly speed up development when used
appropriately. However, AI-generated code is still **your responsibility** — you own it, you maintain it, you debug it.

### Core Principles

1. **Use AI for code you understand and can review.** If you can't explain what the code does, don't commit it. AI is
   a tool to write code faster, not a replacement for understanding.

2. **Never trust AI blindly on critical logic.** Security-sensitive code (encryption, key management, cryptography)
   requires thorough reading and understanding. AI can make subtle mistakes that create vulnerabilities.

3. **Verify before committing.** Always review AI-generated code for:
   - Correctness (does it actually do what you need?)
   - Consistency (does it follow project conventions?)
   - Security (does it introduce vulnerabilities?)
   - Completeness (did it miss edge cases?)

### When AI Works Well

- **Boilerplate and repetitive code**: CRUD operations, test cases, struct definitions
- **Syntax you don't remember**: Regex patterns, SQL queries, unfamiliar APIs
- **Documentation**: Comments, docstrings, README sections
- **Exploring approaches**: "How would I implement X?" as a starting point

### Practical Tips

- **Give context**: The more context you provide (existing code, conventions, requirements), the better the output
- **Iterate**: First drafts are rarely perfect — refine through follow-up prompts
- **Test immediately**: Run the code before assuming it works

---

## Go Coding Conventions

> This section contains patterns that apply to Go backend services in general. See [Project-Specific Guidelines](#project-specific-guidelines) for json-keys-specific patterns.

### File Naming

| Type     | Pattern                   | Example                                |
| -------- | ------------------------- | -------------------------------------- |
| Services | `{operation}.go`          | `jwkGen.go`, `claimsSign.go`           |
| Handlers | `grpc.{resource}.go`      | `grpc.jwkGet.go`, `grpc.claimsSign.go` |
| DAO      | `{source}.{operation}.go` | `pg.jwkSelect.go`, `pg.jwkInsert.go`   |
| Tests    | `{file}_test.go`          | `jwkGen_test.go`                       |
| Mocks    | Generated in `mocks/`     | `mocks/mocks.go`                       |

### Function Naming

```go
// Constructors: New[Type]
func NewJwkGen(deps Dependencies) *JwkGen

// Main business method: Exec
func (s *JwkGen) Exec(ctx context.Context, req *Request) (*Response, error)

// gRPC handlers: Method name matches proto service
func (h *Handler) JwkGet(ctx context.Context, req *protogen.JwkGetRequest) (*protogen.JwkGetResponse, error)

// Private helpers: camelCase
func (s *service) validateInput(req *Request) error
```

### Import Organization

Imports are organized in sections (enforced by `gci`):

```go
import (
    // 1. Standard library
    "context"
    "errors"
    "fmt"

    // 2. External packages
    "google.golang.org/grpc"
    "go.opentelemetry.io/otel"

    // 3. a-novel-kit packages
    "github.com/a-novel-kit/golib/otel"

    // 4. Project packages
    "github.com/a-novel/service-json-keys/v2/internal/dao"

    // 5. Local (embed, etc.)
    _ "embed"
)
```

### Service Pattern

Every service follows this structure:

```go
// 1. Define interfaces for dependencies
type MyServiceRepository interface {
    GetData(ctx context.Context, id string) (*Data, error)
}

// 2. Define request/response structs
type MyServiceRequest struct {
    ID string `validate:"required,uuid"`
}

// 3. Define the service struct
type MyService struct {
    repository MyServiceRepository
}

// 4. Constructor
func NewMyService(repository MyServiceRepository) *MyService {
    return &MyService{repository: repository}
}

// 5. Main method with tracing and validation
func (s *MyService) Exec(ctx context.Context, request *MyServiceRequest) (*Response, error) {
    ctx, span := otel.Tracer().Start(ctx, "service.MyService")
    defer span.End()

    // Validate
    if err := validate.Struct(request); err != nil {
        return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
    }

    // Execute logic
    result, err := s.repository.GetData(ctx, request.ID)
    if err != nil {
        return nil, otel.ReportError(span, err)
    }

    return otel.ReportSuccess(span, result), nil
}
```

### Error Handling

```go
// Define package-level errors with Err prefix
var (
    ErrInvalidRequest = errors.New("invalid request")
    ErrNotFound       = errors.New("resource not found")
)

// Wrap errors with context
return nil, fmt.Errorf("fetch key: %w", err)

// Join errors when adding classification
return nil, errors.Join(err, ErrInvalidRequest)

// Check errors with errors.Is
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

### Handler Pattern (gRPC)

```go
type MyHandler struct {
    protogen.UnimplementedMyServiceServer

    service MyServiceInterface
}

func NewMyHandler(service MyServiceInterface) *MyHandler {
    return &MyHandler{service: service}
}

func (h *MyHandler) MyMethod(ctx context.Context, req *protogen.MyRequest) (*protogen.MyResponse, error) {
    ctx, span := otel.Tracer().Start(ctx, "handler.MyMethod")
    defer span.End()

    // Parse and validate request
    id, err := uuid.Parse(req.GetId())
    if err != nil {
        _ = otel.ReportError(span, err)
        return nil, status.Error(codes.InvalidArgument, "invalid id")
    }

    // Call service
    result, err := h.service.Exec(ctx, &ServiceRequest{ID: id})
    if errors.Is(err, ErrNotFound) {
        _ = otel.ReportError(span, err)
        return nil, status.Error(codes.NotFound, "resource not found")
    }
    if err != nil {
        _ = otel.ReportError(span, err)
        return nil, status.Error(codes.Internal, "internal error")
    }

    // Return response
    return &protogen.MyResponse{...}, nil
}
```

### Dependency Injection

All services receive dependencies via constructor:

```go
// Good: Dependencies injected
func NewService(repo Repository, logger Logger) *Service {
    return &Service{repo: repo, logger: logger}
}

// Bad: Global dependencies
func NewService() *Service {
    return &Service{repo: globalRepo}
}
```

### Validation

Validate at service layer entry, not in handlers:

```go
func (s *Service) Exec(ctx context.Context, req *Request) (*Response, error) {
    // Validate here
    if err := validate.Struct(req); err != nil {
        return nil, errors.Join(err, ErrInvalidRequest)
    }
    // Business logic...
}
```

### Database Conventions

#### Entity Naming

- Tables: `snake_case` plural (e.g., `jwks`, `key_metadata`)
- Columns: `snake_case` (e.g., `created_at`, `expires_at`)

#### Bun ORM Tags

```go
type Jwk struct {
    bun.BaseModel `bun:"table:jwks"`

    ID        uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
    Payload   []byte     `bun:"payload,notnull"`
    ExpiresAt *time.Time `bun:"expires_at"`
    CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp"`
}
```

#### Embedded SQL (PostgreSQL DAO)

For PostgreSQL operations, DAO files use **raw SQL queries in external `.sql` files**, not Bun's query builder. This avoids typical ORM pitfalls (unexpected query generation, N+1 problems, abstraction leaks) and gives full control over the executed SQL:

```go
//go:embed pg.jwkSelect.sql
var jwkSelectQuery string

func (dao *JwkSelect) Exec(ctx context.Context, req *Request) (*Jwk, error) {
    var result Jwk
    err := dao.db.NewRaw(jwkSelectQuery, req.ID).Scan(ctx, &result)
    return &result, err
}
```

Bun is used only for connection management and result scanning — the query itself is always explicit SQL.

---

## Protocol Buffers

The gRPC API is defined using Protocol Buffers in `internal/models/proto/`. These files are the source of truth for all available endpoints, request/response schemas, and service definitions.

### Directory Structure

```
internal/models/proto/
├── jwk.proto           # Common JWK message types
├── jwk_get.proto       # JwkGetService definition
├── jwk_list.proto      # JwkListService definition
├── claims_sign.proto   # ClaimsSignService definition
└── status.proto        # StatusService definition
```

### Editing Proto Files

When adding or modifying proto files:

1. Define your messages and services in `.proto` files
2. Run `make generate` to regenerate Go code
3. Run `make lint-proto` to validate the proto files

### Code Generation

Proto files are compiled to Go code using `buf`:

```bash
# Generate Go code from proto files
make generate

# Lint proto files
make lint-proto

# Format proto files
make format-proto
```

Generated code lives in `internal/handlers/protogen/` and should **never be edited manually**.

### Buf Configuration

The project uses [Buf](https://buf.build/) for proto management. Configuration is in `buf.yaml` and `buf.gen.yaml`.

**Resources:**

- [Buf Documentation](https://buf.build/docs/)
- [Protocol Buffers Language Guide](https://protobuf.dev/programming-guides/proto3/)
- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)

---

## Testing

### Running Tests

```bash
make test       # All tests
make test-unit  # Go unit tests only
make test-pkg   # Package integration tests
```

### Table-Driven Tests

All tests use the **table-driven pattern**: test cases are defined as a slice of structs, each describing inputs, expected outputs, and mock behavior. This approach:

- **Centralizes test logic** — the assertion code is written once and reused for all cases
- **Makes adding cases trivial** — just append a new struct to the slice
- **Improves readability** — each case is self-documenting with a descriptive name
- **Enables parallel execution** — independent cases can run concurrently

```go
testCases := []struct {
    name      string
    request   *Request
    expect    *Response
    expectErr error
}{
    {name: "Success", request: &Request{ID: "123"}, expect: &Response{...}},
    {name: "Error/NotFound", request: &Request{ID: "bad"}, expectErr: ErrNotFound},
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        // test logic here
    })
}
```

### Test Case Naming

Use descriptive names that indicate the scenario:

| Pattern           | Usage                               |
| ----------------- | ----------------------------------- |
| `Success`         | Happy path                          |
| `Success/Variant` | Happy path with specific conditions |
| `Error/Reason`    | Expected failure with cause         |

Examples: `"Success"`, `"Success/WithExpiredKey"`, `"Error/InvalidInput"`, `"Error/NotFound"`

### Parallel Execution

Use `t.Parallel()` to run tests concurrently, reducing total test time:

```go
func TestMyService(t *testing.T) {
    t.Parallel() // Run this test in parallel with other top-level tests

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel() // Run subtests in parallel with each other
            // ...
        })
    }
}
```

**When to use:** Services and handlers tests (with mocks) — these have no shared state.

**When to omit:** DAO tests wrapped in transactions — isolation is managed by the transaction, not parallelism.

### DAO Tests (Real Database)

DAO tests run against a real PostgreSQL instance to verify actual SQL behavior. Each test case is wrapped in a **transaction that rolls back automatically**, ensuring:

- Tests don't pollute each other's data
- No manual cleanup required
- Database state is reset between cases

```go
func TestJwkSelect(t *testing.T) {
    repository := dao.NewJwkSelect()

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
                // 1. Insert fixtures (test-specific data)
                // 2. Run the DAO method under test
                // 3. Assert results
                // Transaction auto-rollbacks after this function returns
            })
        })
    }
}
```

**Fixtures** are defined in the test case struct alongside expected inputs/outputs, then inserted at the start of the transactional callback.

### Service/Handler Tests (Mocks)

Services and handlers are tested with **mocked dependencies**, making tests fast and deterministic. Mock behavior is defined per test case:

```go
testCases := []struct {
    name           string
    request        *Request
    repositoryMock *repositoryMockSetup // Define expected mock behavior
    expect         *Response
    expectErr      error
}{...}
```

Mocks are generated by mockery. After adding or modifying interfaces:

```bash
make generate
```

---

## CI/CD Pipeline

### Continuous Integration

The CI pipeline runs on every push and pull request:

```
┌─────────────────────────────────────────────────┐
│                 Generate Check                  │
│  • Verify go generate is up-to-date             │
│  • Verify buf generate is up-to-date            │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                      Lint                       │
│  • golangci-lint (Go)                           │
│  • buf lint (Proto)                             │
│  • ESLint, Prettier (Node config files)         │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                      Test                       │
│  • Go unit tests with coverage                  │
│  • Integration tests                            │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                      Build                      │
│  • Docker images (database, grpc, migrations)   │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                     Reports                     │
│  • Go Report Card                               │
│  • Codecov coverage                             │
└─────────────────────────────────────────────────┘
```

### Deployment Strategy

We use **snapshot-based publishing** rather than true continuous delivery. Every commit triggers a build, but deployments are environment-specific:

| Source                    | Environment     | Description                                 |
| ------------------------- | --------------- | ------------------------------------------- |
| Feature branch commits    | **Feature env** | Isolated per-branch environment for testing |
| Untagged `master` commits | **Staging**     | Pre-production validation                   |
| Tagged versions           | **Production**  | Stable release snapshots                    |

This approach eliminates the need for a `develop` branch, preventing divergence issues between long-lived branches. All work flows directly from feature branches to `master`.

**Artifacts published on each commit:**

- Docker images (tagged with commit SHA or version)
- Go modules

### Publishing a Release

To publish a new production release, use the publish scripts:

```bash
# Patch release (bug fixes): 2.2.1 → 2.2.2
pnpm publish:patch

# Minor release (new features): 2.2.1 → 2.3.0
pnpm publish:minor

# Major release (breaking changes): 2.2.1 → 3.0.0
pnpm publish:major
```

This will:

1. Bump version in package.json
2. Update version references in documentation
3. Commit and tag the release
4. Push to `master` with the new tag

The CI pipeline then builds and publishes all artifacts with the version tag.

### Docker Images

| Image         | Purpose                    |
| ------------- | -------------------------- |
| `database`    | PostgreSQL with extensions |
| `migrations`  | Schema migration runner    |
| `grpc`        | gRPC API server            |
| `rotate-keys` | Key rotation job           |
| `standalone`  | All-in-one (dev only)      |

---

## Project-Specific Guidelines

> This section contains patterns specific to this JSON Keys service.

### Master Key Encryption

Private JWKs are stored encrypted using AES-GCM with a master key. The master key is:

- Loaded from `APP_MASTER_KEY` environment variable
- Stored in context via `lib.NewMasterKeyContext()`
- Used to encrypt/decrypt private key payloads before storage/retrieval

**Critical:** The master key should **never** be rotated unless absolutely necessary — changing it will permanently lose access to all existing encrypted keys.

### JWK Lifecycle

Keys go through the following states:

1. **Generated**: New key created with expiration time
2. **Active**: Key is used for signing
3. **Expired**: Key past expiration, still valid for verification
4. **Deleted**: Key removed from database

### Key Configuration

Key types and their settings are defined in `internal/config/jwks.config.yaml`:

```yaml
auth:
  algorithm: ES256
  rotation: 720h
  expiry: 8760h
```

Each key usage (e.g., `auth`) can have different algorithms, rotation schedules, and expiry times.

### Key Rotation

The `rotate-keys` job (`cmd/rotate-keys/main.go`) handles automatic key rotation:

- Generates new keys when current ones approach expiration
- Deletes keys past their retention period
- Should be run as a scheduled job (cron, Kubernetes CronJob, etc.)

### gRPC Services

| Service             | Purpose                    |
| ------------------- | -------------------------- |
| `StatusService`     | Health and status checks   |
| `JwkGetService`     | Retrieve single key by ID  |
| `JwkListService`    | List keys by usage         |
| `ClaimsSignService` | Sign JWT claims with a key |

### Go Client Package

Other services integrate with json-keys via the `pkg/` package:

```go
import jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

// Create client
client, err := jkpkg.NewClient("<grpc-address>")

// Sign claims
token, err := client.ClaimsSign(ctx, &jkpkg.ClaimsSignRequest{
    Usage:   "auth",
    Payload: claimsPayload,
})

// Verify claims
verifier := jkpkg.NewClaimsVerifier[MyClaims](client)
claims, err := verifier.VerifyClaims(ctx, &jkpkg.VerifyClaimsRequest{
    Usage:       "auth",
    AccessToken: token.GetToken(),
})
```

---

## Appendix

### A. Key Go Dependencies

| Package                        | Purpose          |
| ------------------------------ | ---------------- |
| `google.golang.org/grpc`       | gRPC framework   |
| `google.golang.org/protobuf`   | Protocol Buffers |
| `github.com/uptrace/bun`       | PostgreSQL ORM   |
| `github.com/a-novel-kit/jwt`   | JWT handling     |
| `github.com/a-novel-kit/golib` | Shared utilities |
| `go.opentelemetry.io/otel`     | Observability    |
| `golang.org/x/crypto`          | Cryptography     |
| `github.com/stretchr/testify`  | Testing          |

### B. Environment Variables

| Variable         | Description                            | Required |
| ---------------- | -------------------------------------- | -------- |
| `GRPC_PORT`      | Port for gRPC server                   | Yes      |
| `POSTGRES_DSN`   | PostgreSQL connection string           | Yes      |
| `APP_MASTER_KEY` | Master key for encrypting private keys | Yes      |

---

## Questions?

If you have questions or run into issues:

- Open an issue at https://github.com/a-novel/service-json-keys/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
