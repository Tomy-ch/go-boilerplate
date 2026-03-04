# Architectural Rules

This document defines the **non-negotiable architectural rules** for this repository.

These rules must be followed by both **human developers and AI agents**.

Violating these rules may break the architectural integrity of the system.

## Layer Dependency Rules

Dependencies must always point **toward the inner layers**.

### Allowed Dependencies

```txt
controller → usecase
usecase → domain
infrastructure → domain
```

### Forbidden Dependencies

```txt
domain → infrastructure
domain → controller
usecase → controller
```

The **domain layer must remain the most independent layer** in the system.

### Rationale

This rule protects the domain model from being coupled to frameworks or infrastructure.

## Generated Code Rules

Certain files are **generated automatically**.

These files must **never be edited manually**.

### Generated Files

Examples include:

- OpenAPI generated server code
- sqlc generated query bindings
- mock generated files

### Rule

Generated files must always be **fully reproducible** from their source definitions.

If a generated file needs to change, modify the **source definition** instead.

Examples:

|Generated Code|Source of Truth|
|---|---|
|OpenAPI server code|OpenAPI specification|
|SQL bindings|SQL query files|
|Mocks|Interface definitions|

## OpenAPI-first

API changes must **always begin with the OpenAPI specification**.

```txt
OpenAPI
  ↓
oapi-codegen
  ↓
Server Interface
  ↓
Handler Implementation
```

### OpenAPI-first Rules

- Do not implement handlers before defining the API contract.
- Generated API interfaces must not be modified manually.

The OpenAPI specification is the **single source of truth for the API**.

## Database Migration

Database schema changes must follow strict migration rules.

### Database Migration Rules

- Existing migration files must **never be modified**
- Migrations are **append-only**
- Schema changes must always begin with a migration

### Typical Flow

```txt
Migration
↓
Schema change
↓
SQL query update
↓
sqlc regeneration
```

This ensures that database history remains reproducible.

## Domain Layer Constraints

The domain layer must remain **pure and independent**.

The following actions are **not allowed** in the domain layer.

### Forbidden in Domain

- External I/O
- Database access
- Environment variable access
- Framework dependencies
- Logging
- HTTP logic

### Allowed in Domain

- Entities
- Value objects
- Domain services
- Business rules
- Repository interfaces

## Infrastructure Implementation Rules

Infrastructure components implement domain interfaces.

Rules:

- Infrastructure must not contain domain logic
- Infrastructure must depend on domain interfaces
- Infrastructure may access external systems

Examples:

- database adapters
- external API clients
- repository implementations

## Layer Responsibility Rules

Each layer has a specific responsibility.

### Controller

Responsible for:

- HTTP transport
- request validation
- error translation

Controllers must **not contain business logic**.

### Usecase

Responsible for:

- application workflows
- transaction boundaries
- coordination of domain logic

Usecases should avoid direct infrastructure dependencies.

## AI Agent Rules

AI-generated code must follow all architectural constraints.

AI agents must:

- respect layer boundaries
- follow OpenAPI-first development
- treat SQL files as contracts
- avoid modifying generated files

Before generating code, AI agents should consult:

```txt
architecture.md
development-flow.md
```

## Summary

These rules exist to ensure:

- architectural consistency
- maintainable code structure
- reproducible builds
- safe collaboration between humans and AI tools
