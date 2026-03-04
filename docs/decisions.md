# Architectural Decisions

This document explains the **technical choices made in this boilerplate**.

The goal is not to claim that these technologies are universally superior, but to clarify **why they were chosen for this architecture**.

These decisions are guided by the following design goals.

## Design Goals

This boilerplate prioritizes:

- Maintainability
- Structural safety
- Type safety
- Replaceable infrastructure
- Long-term operability

Performance and minimal abstraction are **not the primary goals** of this template.

## Why Onion Architecture

### Why Onion Architecture - Intent

Separate business logic from infrastructure and framework dependencies.

### Why Onion Architecture - Decision

The project adopts a **Pragmatic Onion Architecture**.

This structure enforces dependency direction:

```txt
controller → usecase → domain
                     ↑
              infrastructure
```

The domain layer remains independent of external systems.

### Why Onion Architecture - Benefits

- Clear separation of responsibilities
- Improved testability
- Replaceable infrastructure
- Stable domain core

### Why Onion Architecture - Alternatives Considered

#### Layered MVC

Simpler but tends to mix domain and infrastructure logic.

#### Clean Architecture

Very similar in concept, but often introduces additional abstraction layers.

The chosen approach is a **simplified pragmatic version**.

## Why OpenAPI-first

### Why OpenAPI-first - Intent

Define API contracts explicitly before implementation.

### Why OpenAPI-first -  Decision

API specifications are written using **OpenAPI**, and server code is generated using `oapi-codegen`.

### Why OpenAPI-first -  Benefits

- Clear API contracts
- Type-safe request/response structures
- Consistency between backend and frontend
- Automatic API documentation

### Why OpenAPI-first -  Alternatives Considered

#### Code-first API

Generating OpenAPI from code can lead to unclear API contracts.

#### GraphQL-first

GraphQL introduces additional complexity and is not always necessary for typical backend services.

## Why SQL-first

### Why SQL-first - Intent

Treat SQL as a **first-class contract** rather than hiding it behind an ORM.

### Why SQL-first - Decision

Queries are written directly in SQL and compiled using `sqlc`.

### Why SQL-first - Benefits

- Full control over queries
- Clear performance characteristics
- Explicit data access patterns

### Why SQL-first - Alternatives Considered

#### Full ORM

ORM frameworks often obscure query behavior and performance.

#### Query Builder

Query builders reduce SQL visibility and can introduce additional complexity.

## Why sqlc

### Why sqlc - Intent

Combine explicit SQL with **type-safe Go code**.

### Why sqlc - Decision

`sqlc` is used to generate Go code from SQL queries.

### Why sqlc - Benefits

- Compile-time type safety
- Explicit SQL definitions
- Minimal runtime abstraction

### Why sqlc - Alternatives Considered

#### GORM

Provides convenience but introduces ORM abstraction and hidden queries.

#### Ent

Schema-first approach that requires a different workflow.

## Why Echo

### Why Echo - Intent

Provide a lightweight and predictable HTTP framework.

### Why Echo - Decision

The project uses **Echo** for HTTP routing and middleware.

### Why Echo - Benefits

- Simple and explicit middleware system
- Minimal abstraction
- Good performance characteristics

### Why Echo - Alternatives Considered

#### Gin

Very similar, but Echo provides slightly cleaner middleware composition.

#### Chi

Excellent router, but Echo provides a more complete framework experience.

## Why Fx

### Why Fx - Intent

Provide structured dependency injection and application lifecycle management.

### Why Fx - Decision

The project uses **Uber Fx** as the dependency injection container.

### Why Fx - Benefits

- Explicit dependency wiring
- Lifecycle management
- Structured module composition

### Why Fx - Alternatives Considered

#### Manual dependency wiring

Works well for small systems but becomes difficult to maintain as the system grows.

#### Google Wire

Compile-time DI but lacks runtime lifecycle management.

## Future Evolution

These decisions are **not immutable**.

Technologies may change if:

- the ecosystem evolves
- better tools become available
- architectural constraints change

However, changes should maintain the **core design goals of this template**.
