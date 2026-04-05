# Boundary Packages

Overview: `internal/usecase/boundary` is a set of packages that define the **boundary with external systems as seen from the Usecase layer**.  
This layer has the responsibility of **abstracting external dependencies such as authentication, transactions, time, cryptography, and jobs, and defining them as interfaces (contracts)**.

In Onion Architecture, it functions as a **protective wall to block dependencies from the inner layer (Usecase) to the outer layer (Infrastructure)**.

## Essence of this layer

boundary is not just a collection of interfaces.

The purposes are the following three:

- A layer that converts external dependencies into "named concepts"
- A layer that explicitly defines extension points of the system
- A structural guard to enforce dependency direction

## Responsibilities

- Abstraction of external dependencies (DB / authentication / time / cryptography / batch, etc.)
- Enforcing a structure where Usecase does not depend on external implementations
- Providing implementation swap points via DI
- Providing mock swap points during testing
- Assigning meaning to errors (application-level granularity)

## Dependencies

+++mermaid
flowchart TB
    A["Controller"] --> B["Usecase"] --> C["Boundary (interface)"] --> D["Infrastructure (implementation)"]
+++

- Usecase depends only on boundary
- infrastructure implements boundary
- boundary does not depend on anything (exception: apperror / util)

## Responsibilities of sub-packages

### auth

Responsibilities:

- Input of authentication information (Credential)
- Representation of authentication result (Authn)
- Abstraction of authentication processing (Authenticator)

Characteristics:

- normalization of subject (trim)
- encapsulation of UUID conversion (HasID / ID)
- retention of scope / claims (for authorization / UI control)

Design intention:

- Represent the state of "authenticated" with types
- Push token parsing logic outward

### tx

Responsibilities:

- Control of transaction boundaries

Characteristics:

- Scope control via `Manager.Do`
- Value return via `DoWithResult`

Design intention:

- Make Usecase aware of the "existence of transactions"
- Completely hide DB dependencies

### clock

Responsibilities:

- Retrieval of current time

Design intention:

- Ensure testability of time-dependent logic
- Reproducibility of TTL / expiration

### security

Responsibilities:

- Hashing and comparison of passwords, etc.

Design intention:

- Replaceability of cryptographic algorithms
- Encapsulation of bcrypt / argon2, etc.

### job

Responsibilities:

- Abstraction of job execution

Characteristics:

- Separation of Job / Runner / State
- CLI / batch execution infrastructure

Design intention:

- Abstract execution units and eliminate implementation dependencies
- Testable batch infrastructure

## Design Rules

### MUST

- Define interfaces only (implementation is prohibited)
- Do not depend on external libraries
- Accept context (if I/O exists)
- Wrap errors with `apperror`
- Treat DTOs as immutable (no side effects)

### SHOULD

- Interfaces should have single responsibility
- Naming should be role-based (Manager, Authenticator, etc.)
- Assume mock generation (mockgen)
- DTOs should be minimal

### MUST NOT

- Depend on Echo / sqlc / AWS SDK / HTTP, etc.
- Expose infrastructure types
- Write implementation logic
- Pollute Domain (reverse dependency)

## Anti-patterns

### ❌ Writing implementation in boundary

+++go
func (a *authenticator) Authenticate(...) { ... }
+++

→ NG  
boundary is contracts only

### ❌ Leaking infra types

+++go
type Authenticator interface {
    Authenticate(...) (*jwt.Token, error)
}
+++

→ NG  
external dependency is leaked

### ❌ Including domain logic

→ NG  
that is the responsibility of Domain

### ❌ Putting everything into boundary

→ NG  
only "boundary with external dependencies"

## Design decision criteria

### Should this be placed in boundary?

YES conditions:

- Depends on external systems (DB / API / time / authentication)
- There is a possibility of implementation replacement
- Needs to be mocked during testing
- Directly called by Usecase

NO conditions:

- Pure business logic → Domain
- Simple utility → pkg
- Logic inside Usecase → usecase

## Error Design

- boundary defines "meaningful errors"
- Wrap and return apperror

Examples:

- Unauthenticated
- Invalid arguments
- Invalid ID

## Relationship with DI (uber/fx)

- boundary is not provided (only abstraction)
- infrastructure provides implementation
- usecase receives interface

+++go
type UserUsecase struct {
    auth auth.Authenticator
    tx   tx.Manager
}
+++

## Test Strategy

- Use mocks in Usecase tests
- Completely eliminate external dependencies

+++go
mockAuth.EXPECT().
Authenticate(gomock.Any(), gomock.Any()).
Return(&auth.Authn{...}, nil)
+++

## Why this structure

With this structure:

- Implementation changes (DB / authentication method) do not affect
- Tests become fast and stable
- Dependencies do not break (can be enforced by lint)
- Even if AI writes code, the structure does not collapse

## What happens if this layer breaks

- Usecase directly depends on infrastructure
- Becomes untestable
- Cannot be replaced
- Becomes spaghetti

## Recommended Development Flow

1. Define interface in boundary
2. Implement Usecase (based on interface)
3. Implement in infrastructure
4. Bind with fx
5. Test with mock
