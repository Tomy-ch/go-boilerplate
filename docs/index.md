# Documentation

English | [日本語](ja/index.ja.md)

This directory contains the architecture and development documentation for the **go-boilerplate** project.

These documents describe the design philosophy, architectural rules, and development workflows used in this repository.

The documentation is intended for both **human developers** and **AI agents**.

## Documentation Overview

|Document|Description|
|--------|--------|
|[architecture.md](architecture.md)|High-level architecture and layer responsibilities|
|[rules.md](rules.md)|Architectural constraints that must not be violated|
|[development-flow.md](development-flow.md)|Standard development workflows|
|[decisions.md](decisions.md)|Rationale behind architectural and technology choices|

## Recommended Reading Order

For new developers:

```txt
architecture.md
↓
development-flow.md
↓
rules.md
↓
decisions.md
```

For maintainers and contributors:

```txt
architecture.md
↓
rules.md
↓
development-flow.md
↓
decisions.md
```

For AI agents:

```txt
rules.md
↓
architecture.md
↓
development-flow.md
↓
decisions.md
```

## Key Concepts

The boilerplate is built around several core principles.

### Onion Architecture

The system follows a layered architecture:

```txt
controller → usecase → domain ← infrastructure
```

Dependencies always point inward.

### OpenAPI-first Development

API contracts are defined using OpenAPI.

Implementation must follow the contract definition.

```txt
OpenAPI
↓
Code generation
↓
Handler implementation
↓
Usecase implementation
```

### SQL-first Data Access

Data access is designed around SQL rather than ORM abstractions.

```txt
SQL
↓
sqlc
↓
Type-safe Go code
```

### Structural Safety

This repository prioritizes **structural safety over convention**.

Instead of relying on implicit rules or manual review, the architecture enforces safety through:

- Layer separation
- Code generation
- CI checks
- Lint rules

## AI-Assisted Development

This repository is designed to work safely with AI-assisted development tools.

Constraints are intentionally introduced to prevent architectural violations.

AI agents should always consult:

```txt
rules.md
architecture.md
```

before generating code.

## Relationship With Other Documentation

High-level overview:

```txt
README.md
   ↓
docs/index.md
   ↓
architecture / rules / development-flow
```

`README.md` provides a quick overview of the project,  
while the `docs/` directory contains detailed design documentation.

## Contribution Guidelines

When contributing to the repository:

1. Follow the architectural rules defined in `rules.md`
2. Follow the workflows described in `development-flow.md`
3. Ensure changes remain consistent with the architecture described in `architecture.md`

If architectural changes are required, update the corresponding documentation.

## Philosophy

This boilerplate aims to provide a **safe and maintainable backend starting point**.

The goal is not to enforce a single "correct" architecture, but to provide a **well-structured baseline** that teams can adapt to their needs.
