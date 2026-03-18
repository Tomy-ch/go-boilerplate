# Infrastructure Layer Guide (`internal/infrastructure`)

## Responsibility

The Infrastructure layer is responsible for **implementing access to external technologies** (DB, external APIs, authentication, security, etc.).

This layer has the following responsibilities:

- Implement external I/O (RDB / API / authentication / system)
- Implement interfaces defined by the Domain
- Encapsulate technical details (connection, retry, drivers, logging, etc.)
- Normalize errors
- Provide Observability (logging / tracing)

Upper layers (Domain / Usecase) **must not be aware of Infrastructure implementation details**.

## Position in Onion Architecture

```txt
Domain
  ↑
Usecase
  ↑
Infrastructure
```

- Domain / Usecase define abstractions only
- Infrastructure provides concrete implementations

## Dependencies

```txt
Domain ← Infrastructure (implementation)
Usecase ← Infrastructure (usage)
```

- Infrastructure depends on Domain
- Domain / Usecase must not depend on Infrastructure

## Relationship with Usecase

- Transaction boundaries are managed by the Usecase layer
- Infrastructure must not start transactions
- Transactions are propagated via `context.Context`

```txt
Usecase (starts Tx)
  ↓
Repository / QueryService
  ↓
driver (uses Tx)
```

## Error Handling

Infrastructure must not return raw external errors.  
Instead, it must **convert them into application-level errors**.

Examples:

- PostgreSQL errors → `pgerror.NormalizeError`
- External API errors → converted to `apperror`

## Observability

Infrastructure provides the following observability features:

- SQL / external I/O logging
- OpenTelemetry tracing
- Execution time measurement (e.g., slow queries)

Typically implemented via wrappers such as `loggingdb`.

## Prohibited Practices

The following must not be done in the Infrastructure layer:

- Implement business logic
- Branch on Domain rules
- Make Usecase-level decisions
- Introduce HTTP / framework-dependent code
- Start transactions

## Implementation Rules

- Use `sqlc` for SQL execution
- Do not implement search logic in Repository (use QueryService)
- Do not use driver directly; use it via loggingdb
- Always propagate `context`
- Always normalize external errors

## Directory Structure

```txt
internal/infrastructure
 ├ auth/
 ├ rdb/
 ├ security/
 └ system/
```

## Subsystems

### Authentication

- Token validation
- Extraction of authentication information

→ See [auth/README.md](./auth/README.md)

### RDB Access

- Repository / QueryService
- Type-safe SQL execution via sqlc
- Transaction management via driver
- Logging / tracing via loggingdb
- PostgreSQL error normalization

→ See [rdb/README.md](./rdb/README.md)

### Security

- Encryption / hashing
- Token generation

→ See [security/README.md](./security/README.md)

### System

- Clock (time management)
- ID generation
- System utilities

→ See [system/README.md](./system/README.md)

## Test Strategy

- Integration tests using a real database
- State isolation via transaction rollback
- Use `testkit`

```txt
Real DB + rollback + no parallel execution (Tx serialized)
```

## Design Principles Summary

### 1. Encapsulation of Technical Details

DB / API / authentication / security  
→ encapsulated within Infrastructure

### 2. Dependency Inversion

Domain defines interfaces  
Infrastructure implements them

### 3. Separation of Responsibilities

```txt
Persistence → Repository
Query       → QueryService
```

### 4. Transaction Management

Usecase manages transactions  
Infrastructure does not participate

### 5. Unified Error Handling

External errors → application errors

### 6. Observability

logging / tracing / metrics
