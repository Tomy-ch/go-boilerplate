# Architecture

## Overview

This project provides a backend architecture template for Go applications based on three primary goals:

- **Contract-driven development**
- **Type safety**
- **Clear layer separation**

The template combines the following architectural approaches:

- Pragmatic Onion Architecture
- OpenAPI-first development
- SQL-first data access
- Dependency Injection
- Structural safety enforced by CI

These components form a backend foundation that prioritizes **maintainability, predictability, and structural safety**.

This architecture is not intended to fit every type of system.  
It is particularly effective for **business systems and long-lived backend services**.

## Architectural Principles

The template is built on several design principles.

## Contract-first API

API contracts are defined using **OpenAPI**.

Code generation ensures that the implementation stays consistent with the API specification.

Typical workflow:

```txt
OpenAPI specification
↓
Code generation
↓
Handler implementation
↓
Usecase implementation
```

Generated code must **never be edited manually**.

### Dependency Inversion

Dependencies always point **toward the inner layers**.

```txt
controller → usecase → domain
                     ↑
              infrastructure
```

Key ideas:

- Inner layers do not depend on outer layers
- Domain remains independent from frameworks
- Infrastructure implements domain interfaces

This rule protects the **stability of the core domain logic**.

### SQL-first Data Access

Data access is designed around **SQL rather than ORM abstractions**.

SQL queries are defined explicitly and compiled into type-safe Go code using `sqlc`.

Benefits:

- Full control over queries
- Compile-time type safety
- Clear performance characteristics

### Structural Safety

This repository prioritizes **structural safety over implicit conventions**.

Instead of relying solely on code review or team discipline, safety is enforced through tools.

Examples:

- Code generation
- Lint rules
- CI validation
- Layer boundaries

These mechanisms help prevent accidental architectural violations.

### Vendor Neutrality

The template avoids strong coupling to specific SaaS platforms or proprietary tooling.

Where possible, the architecture prefers:

- OSS-based tooling
- Replaceable components
- Vendor-neutral integrations

This ensures long-term flexibility.

## System Architecture

The system follows a **Pragmatic Onion Architecture**.

```txt
┌─────────────┐
│ Controller  │
└──────┬──────┘
       ↓
┌─────────────┐
│   Usecase   │
└──────┬──────┘
       ↓
┌─────────────┐
│   Domain    │
└──────┬──────┘
       ↑
┌─────────────┐
│Infrastructure│
└─────────────┘
```

Characteristics:

- Outer layers depend on inner layers
- Domain is the most stable layer
- Infrastructure implements domain interfaces

This structure allows the domain logic to remain stable even if external systems change.

## Layer Responsibilities

## Controller

The controller layer handles **HTTP transport concerns**.

Responsibilities:

- HTTP request/response handling
- Input validation
- Error translation
- Calling usecases

Controllers must **not contain business logic**.

### Usecase

The usecase layer implements **application-level logic**.

Responsibilities:

- Application workflows
- Coordination of domain objects
- Transaction boundaries
- Interaction between domain and infrastructure

Usecases orchestrate domain behavior but should avoid low-level infrastructure details.

### Domain

The domain layer represents the **core business logic**.

Responsibilities:

- Entities
- Value objects
- Domain rules
- Repository interfaces

Domain code must remain **pure and independent of external frameworks**.

### Infrastructure

The infrastructure layer integrates external systems.

Responsibilities:

- Database access
- External service integration
- Repository implementations

Infrastructure components implement interfaces defined by the domain layer.

## Request Flow

A typical request follows this flow:

```txt
HTTP Request
   ↓
Echo Router
   ↓
Controller
   ↓
Usecase
   ↓
Domain
   ↓
Repository Interface
   ↓
Infrastructure
   ↓
Database
```

This structure ensures that:

- HTTP logic stays in controllers
- application orchestration stays in usecases
- business logic stays in domain

## Dependency Injection

This project uses **Uber Fx** as the dependency injection container.

The DI container is responsible for:

- component initialization
- dependency resolution
- lifecycle management

Typical wiring order:

```txt
Repository
 ↓
Usecase
 ↓
Handler
 ↓
Router
```

Using DI helps maintain loose coupling between layers.

## Code Generation

Code generation plays a critical role in this architecture.

The following components rely on generation:

- OpenAPI server interfaces (`oapi-codegen`)
- SQL query bindings (`sqlc`)

Rules:

- Generated code must not be edited manually
- Generated code must be reproducible
- CI checks verify generation consistency

## Project Structure

Key directories:

```txt
cmd/
Application entrypoints

internal/
Application code

database/
SQL queries and migrations

openapi/
API contracts

docs/
Architecture documentation
```

The `internal/` directory contains the layered application code.

## Modular Monolith Strategy

This template assumes a **modular monolith architecture**.

Characteristics:

- single deployable application
- internal module boundaries
- strict layer separation

Microservices are **not the primary design goal**.

However, clear module boundaries make future extraction possible.

## Extensibility

Components under `internal/` are connected using dependency injection.

This allows relatively easy replacement of:

- repositories
- middleware
- infrastructure integrations

The architecture is designed to allow **infrastructure changes without modifying domain logic**.

## AI-Assisted Development

This repository is designed to work safely with AI-assisted development tools.

Architectural constraints help prevent AI-generated code from violating core design rules.

AI agents should always consult:

- `rules.md`
- `architecture.md`

before generating code.

## Non-Goals

This template does not aim to be:

- a microservice framework
- an ultra-low latency architecture
- a universal architecture for all system types

The goal of this project is to provide a **maintainable and structurally safe backend architecture** for typical business applications.
