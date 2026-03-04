# go-boilerplate Agents Documentation

## Project overview

This repository is a lightweight Onion Architecture–based Go boilerplate designed for building RESTful APIs.

It follows a strict layered structure:

controller → usecase → domain  
infrastructure implements domain interfaces

Key principles:

- OpenAPI-first development (contract-driven)
- sqlc-based type-safe database access
- Generated code must not be edited manually
- Business logic must remain inside `internal/`
- Dependencies must always point inward
- Clear separation between domain models and DTOs
- Deterministic and framework-agnostic domain layer

The structure is intentionally strict to ensure:

- Maintainability
- Testability
- Replaceable infrastructure
- Safe AI-assisted development

Do not introduce new architectural patterns unless explicitly instructed.

## Architecture Documentation

This repository contains structured architecture documentation under `docs/`.

AI agents MUST read these documents before making architectural changes.

Recommended reading order:

Required reading:

```txt
docs/architecture.md
docs/rules.md
```

Optional reference:

```txt
docs/development-flow.md
docs/decisions.md
```

Purpose of each document:

|Document|Purpose|
|----------|--------|
|rules.md|Non-negotiable architectural constraints|
|architecture.md|System structure and layer responsibilities|
|development-flow.md|How development tasks must be performed|
|decisions.md|Rationale behind technology choices|

When generating or modifying code, always follow the rules defined in `docs/rules.md`.

## Root Folders

### AI Modification Scope

AI agents are allowed to modify code only in the following directories unless explicitly instructed otherwise:

- `internal/`
- `pkg/`
- `database/`
  - `database/dml/**`
  - `database/migrations/**` (new files only)
- `openapi/`

Do NOT modify other top-level directories (e.g., `cmd/`, `docker/`, `scripts/`, `docs/`, `vendor/`, `makefile`, etc.) unless the user explicitly requests it.

### internal/

Core application code.

- Organized by onion layers (domain, usecase, infrastructure, controller, etc.).
- All business logic must live inside this directory.
- This is the primary location for feature implementation.

### pkg/

Shared utilities and reusable components.

- Framework-agnostic helpers.
- Safe to use across layers (including domain).
- Should not contain business logic specific to a single feature.
- pkg must not depend on infrastructure or framework-specific packages.

### database/

Database-related artifacts.

- Migrations (`database/migrations/`)
- Raw SQL files for sqlc (`database/dml/`)
- Schema management

Source of truth for database structure and queries.

### openapi/

OpenAPI specifications.

- Source YAML files
- Bundled OpenAPI file
- Contract definition for HTTP APIs

All API changes must start here.

## Core Architecture (`internal/` folder)

The `internal/` directory contains all application code, organized following a lightweight Onion Architecture.

Layered structure and responsibilities:

### domain/

Defines core business concepts and contracts.

- Entities
- Value objects
- Domain services
- Repository interfaces
- No framework or infrastructure dependencies allowed
- Only use shared utilities defined under `pkg/` when external libraries are required
  - If time or randomness is required, it must be abstracted via domain interfaces and implemented in outer layers.
- Constructors must not instantiate external dependencies
- Business logic must remain deterministic and side-effect free

This layer must remain pure, independent, and framework-agnostic.

Domain layer must not:

- Access environment variables
- Perform I/O
- Use logging frameworks
- Use context.Context
- Use database clients
- Import infrastructure packages
- Depend on time, randomness, or system state directly

### usecase/

Implements application use cases.

- Orchestrates domain services and business flows
- Depends only on domain interfaces
- Must not depend on infrastructure implementations
- Transforms domain models into application DTOs before returning
- Must not expose domain entities directly to outer layers

Business rules coordination and response shaping belong here.

Usecase is responsible for:

- Transaction boundaries
- Aggregating multiple domain operations
- Mapping errors into application-level errors

### infrastructure/

Provides concrete implementations of domain interfaces.

- Database access (sqlc, repositories)
- External service integrations
- Persistence logic

This layer depends on domain, never the other way around.

### controller/

Represents the outer interface layer (HTTP/API).

- Echo handlers
- OpenAPI-generated server implementations
- Request validation
- Response formatting

The handler remains lightweight, and business logic calls must be completed within the use case.

### di/

Defines dependency wiring using Uber Fx.

- Provider registration
- Module grouping
- Application composition

No business logic should exist here.

### config/

Handles configuration and environment loading.

- Environment variables
- Application settings
- Configuration structs

### apperror/

Defines application-wide error types and mappings.

- Domain errors
- HTTP error translation helpers

### logging/

Provides structured logging utilities (zap-based).

- Logger initialization
- Shared logging helpers

### observability/

Contains telemetry-related components.

- Metrics
- Tracing
- Instrumentation hooks

### integration/

Integration-level components and helpers.

- External system integration logic
- Shared integration utilities

### cli/

CLI entrypoints and Cobra command definitions.

This directory is infrastructure-level and should not contain business logic.
AI agents should not modify this directory unless explicitly instructed.

### system/

System-level utilities and operational helpers.

This directory is considered internal infrastructure.
Do not generate new code here unless explicitly requested.

## Layer Rules (Strict)

- Do NOT bypass layers.
- Domain must not depend on infrastructure.
- Usecase must call domain interfaces only.
- Infrastructure implements domain interfaces.
- Handlers must not contain business logic.

Always search for existing implementations in the same layer before creating new files.
Do NOT introduce new architectural patterns without explicit instruction.

### Finding Related Code

When implementing or modifying functionality, follow the onion architecture flow:

1. Start from the OpenAPI definition:
    - Source files: `openapi/**/*.yaml`
    - Bundled file (single source of truth for generation): `openapi/openapi.gen.yaml`
    - Generated server interfaces: `internal/controller/handler/**/gen/server.gen.go`
    - Generated server types: `internal/controller/handler/**/gen/type.gen.go`

2. Locate the HTTP handler implementation:
    - HTTP handler implementation: `internal/controller/handler/**/*_handler.go`

3. From the handler, follow the usecase call:
    - `internal/usecase/**/*_usecase.go`

4. Check the domain interfaces:
    - `internal/domain/**/*_domain.go`
    - `internal/domain/**/*_repository.go`

5. Locate infrastructure implementations:
    - domain repository implementations: `internal/infrastructure/rdb/repository/**/*_repository.go`
    - query service implementations: `internal/infrastructure/rdb/query_service/**/*_query_service.go`
    - system queries implementations: `internal/infrastructure/rdb/system_query/**/*_system_query.go`
    - sqlc generated files: `internal/infrastructure/rdb/sqlc/gen/*.gen.go`

6. For database queries:
    - Repository sql files: `database/dml/repository/**/*.sql`
    - Query service sql files: `database/dml/query_service/**/*.sql`
    - System query sql files: `database/dml/system_query/**/*.sql`
    - Migration files: `database/migrations/*.sql`

Migration rules:

- Never modify existing migration files.
- Always create a new migration file.
- Follow the existing version naming convention.

### Change Type Guidelines

API Change:

API changes MUST follow this order:

1. Modify OpenAPI source (`openapi/**/*.yaml`)
2. Regenerate code (`make gen-api`)
3. Implement handler
4. Implement usecase

Handlers and usecases MUST NOT be implemented before the OpenAPI contract exists.

Database Change:

- Add migration → update SQL files → regenerate sqlc → update repository → update usecase

Business Logic Change:

- Modify usecase → adjust domain if necessary → ensure repository contract remains unchanged

### Key Technologies

- [echo](https://echo.labstack.com/) - Web framework for building RESTful APIs
- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) - Generate Go server and client code from OpenAPI specs
- [sqlc](https://sqlc.dev/) - Generate type-safe Go code from SQL queries
- [go.uber.org/mock](https://github.com/uber-go/mock) - Mock generation tool for Go interfaces
- [go.uber.org/zap](https://github.com/uber-go/zap) - Fast, structured, leveled logging in Go
- [go.uber.org/fx](https://github.com/uber-go/fx) - Fast, flexible dependency injection framework for Go
- [go.opentelemetry.io/otel](https://github.com/open-telemetry/opentelemetry-go) - OpenTelemetry for distributed tracing and metrics

## Recommended commands

### Generated Code Commands

- `make gen-api` - Generate API code from OpenAPI spec by oapi-codegen and go.uber.org/mock
- `make gen-query` - Generate SQL query code from SQL files by sqlc

### CRITICAL Git Rules for AI Agents

1. NEVER commit directly to `production`, `develop`, `staging`, or any `release/*` branch (e.g., `release/v0.1.0`) - Always create a feature branch for your work from the latest `release/*` branch. Direct commits to these branches are strictly prohibited.

2. Do NOT rebase, squash, or force-push unless explicitly requested by the user. Default behavior should be regular commits and pushes.

3. After amending an existing pull request branch, do NOT push changes automatically.
After making changes to an existing PR branch locally, ask the user before pushing. This lets the user review the updated commits on their machine first. Only push automatically if the user’s instructions explicitly include performing the push.

### Branch Naming

- If an issue number is provided, include it in the branch name.
  Examples: `feature/1234-description`, `bugfix/5678-fix-issue`

- If no issue number exists, generate a descriptive name
  based on the task goal, using hyphen-separated terms.
  Example: `feature/add-authentication-check`

### Git Workflow

```bash
# Create a feature branch (never work directly on protected branches)
git switch -c feature/issue-12345-short-description

# Add changes and commit with a descriptive message
git add .
git commit -m "feat: short description of change"

# Push to remote (first push sets upstream)
git push -u origin feature/issue-12345-short-description

# For subsequent pushes on the same branch
git push
```

**When asked to update an existing PR:**

1. Make the requested changes
2. Stage and commit the changes
3. **STOP and ask the user** before pushing: "変更はローカルにコミット済みです。これらの変更をプルリクエストにプッシュしますか？"

## Code style guidelines

### Code Formatting

Always format code before committing:

```bash
make fix
```

After executing the command, please correct any areas that could not be fixed.

Finally, run `make lint` to check for any errors.

### Do not edit generated files and directlies

- `**/**.gen.go`
- `**/**.sql.go`
- `*_mock.go`
- `**/openapi.gen.yaml`
- `docs/`

## Testing Instructions

### 1. Code Generation (Required Before Testing)

Before writing or modifying tests, always execute:

```sh
    make gen-api
    make gen-query
```

- Always use generated mocks.
- Do NOT create custom mocks inside test files unless explicitly instructed.
- Reuse existing mocks generated by `uber.org/mock`.

### 2. Test Structure Rules

- Tests must be created per function.
- All logical branches must be covered.
- Use table-driven tests where appropriate.
- Subtests must use `t.Run()`.

Parallel execution is required:

```go
    t.Parallel()
```

Tests must be independent and deterministic.

### 3. Naming Convention

- All test case names must be written in Japanese.
- Test names must clearly describe the behavior and branch condition.

Example:

```text
    "正常系_ユーザーが存在する場合"
    "異常系_ユーザーが存在しない場合"
```

### 4. Assertion Rules

- Use `require` for preconditions and fatal checks.
- Use `assert` for value verification.

Example:

```go
    require.NoError(t, err)
    assert.Equal(t, expected, actual)
```

### 5. Coverage Requirement

After writing tests, verify coverage:

```sh
    make test
```

Coverage must not decrease from the current baseline.
New or modified packages must exceed 90% coverage.

If coverage is below 90%:

- Add missing branch tests.
- Re-run tests.
- Do not stop until the requirement is met.

### 6. Generated Files

Do NOT:

- Modify generated mock files
- Modify `*.gen.go`
- Modify `*.sql.go`

Tests must rely only on public interfaces and generated artifacts.

### 7. Architectural Rules in Tests

- Domain tests must not use infrastructure implementations.
- Usecase tests must mock domain repositories.
- Controller tests must mock usecases.
- Do not bypass layers in tests.

Tests must respect the same onion architecture rules as production code.

### Running Tests

`make test` - Run all tests
`make fix` - Auto-format code and fix lint issues

## Protected Documentation

The following file defines architectural and operational constraints:

- `AGENTS.md`

This file must be maintained by humans only.

AI agents must NOT modify this file unless explicitly instructed by a human.

Changes to this file must be intentional and reviewed carefully, as it defines repository-wide development rules.

If the correct implementation location is unclear, prefer modifying an existing file rather than creating a new one.
