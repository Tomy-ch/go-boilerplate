# Development Workflow

This document describes the **standard development workflows** used in this repository.

These workflows ensure that the architecture remains consistent and that generated code stays synchronized with the source definitions.

## API Change Flow

When modifying an API endpoint, the following order **must be followed**.

### API Change Flow Steps

1. Update the OpenAPI specification in `openapi/`
2. Regenerate API code
3. Implement or update the handler
4. Implement or update the usecase logic

```txt
OpenAPI
↓
Code generation
↓
Handler
↓
Usecase
```

### API Change Flow Notes

- OpenAPI is the **source of truth** for the API contract.
- Generated files must **never be edited manually**.

## Database Change Flow

Database changes must follow a strict order to ensure consistency between SQL and generated code.

### Database Change Flow Steps

1. Create a new migration file
2. Update or add SQL queries
3. Regenerate `sqlc` code
4. Update repository implementations if necessary
5. Update usecases if required

```txt
Migration
↓
SQL query update
↓
sqlc generation
↓
Repository update
↓
Usecase update
```

### Database Change Flow Notes

- SQL files are treated as **contracts**.
- `sqlc` generated code must not be edited manually.

## Business Logic Change Flow

Changes to application behavior typically occur in the following order.

### Business Logic Change Flow Steps

1. Update usecase logic
2. Update domain logic if necessary
3. Update repository interfaces if data access changes

```txt
Usecase
↓
Domain
↓
Repository Interface
```

Infrastructure changes are **often unnecessary** unless external integrations or database logic change.

## Code Generation

This project relies on code generation for API and database access.

## API Code Generation

```txt
make gen-api
```

This generates server interfaces and types from the OpenAPI specification.

## SQL Code Generation

```txt
make gen-query
```

This generates type-safe Go code from SQL queries using `sqlc`.

## Generated Code Rules

Generated files are **not intended to be edited manually**.

Rules:

- Do not modify generated files
- Always regenerate after updating source definitions
- CI checks may verify that generated code is up-to-date

If generated code needs to change, modify the **source definition** instead.

Examples:

|Generated code|Source of truth|
|---|---|
|API server code|OpenAPI specification|
|SQL query bindings|SQL files|

## Typical Feature Development Flow

A typical feature implementation follows this order.

```txt
1 Define API (OpenAPI)
2 Implement usecase
3 Implement repository if needed
4 Implement handler
5 Add tests
```

This order ensures that **API contracts and domain behavior remain consistent**.

## CI and Structural Safety

CI checks help maintain architectural consistency.

Typical checks include:

- Code generation verification
- Lint checks
- Build validation

These checks help prevent architectural drift and ensure that the repository remains reproducible.

## AI-Assisted Development

AI tools may assist with implementation, but the development workflow must still be respected.

AI-generated code must follow:

- OpenAPI-first workflow
- SQL-first data access
- Layer separation rules

AI agents should consult:

```txt
architecture.md
rules.md
```

before generating code.

## Summary

This workflow ensures:

- consistent API contracts
- safe database migrations
- predictable code generation
- architectural integrity
