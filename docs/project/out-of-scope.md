# What This Project Intentionally Does NOT Include

## Items Dependent on Company Infrastructure Choices

- Deployment implementation  
  - Only a skeleton is provided: [.github/workflows/deploy-app.yaml](../../.github/workflows/deploy-app.yaml)
- Infrastructure as Code (IaC)
- Observability operational configuration
- Circuit breaker
- Secret rotation
- Rate limiting
  - Intentionally not provided as an in-application (in-memory) limiter
  - In a cloud-native, multi-instance deployment, per-instance in-memory
    counters do not share state and cannot enforce a correct global limit
  - This belongs at the infrastructure edge (API gateway / load balancer /
    reverse proxy / service mesh)
- Scheduled job concurrency control
  - Overlap / multi-instance guarding for scheduled jobs (k8s CronJob
    `concurrencyPolicy`, advisory locks) is left to the scheduler
  - No application-level mutual exclusion is provided because the bundled
    jobs are already concurrency-safe by design: `outbox-gc` and
    `idempotency-gc` are age-predicate, idempotent batch deletes,
    `usercount` is read-only, and the outbox relay claims rows with
    `FOR UPDATE SKIP LOCKED`
  - If you require strict single-run semantics, set
    `concurrencyPolicy: Forbid` at the scheduler

## Items Strongly Dependent on Domain Requirements

- Audit logging
- RBAC / authorization model
- Session management
- Password policy  
  - No in-repo credential store is provided: authentication is delegated to an external OIDC / JWT (Bearer) IdP, so this service holds no passwords. See [docs/design/auth.md](../design/auth.md).
- Data retention policy  
  - Soft delete is provided as a sample
- Encryption for PII storage

## Items Expected to Be Implemented by Users

- Authentication mechanisms (JWT, Cookie, OAuth2, etc.)  
  - A sample implementation is provided, designed to be extensible  
    - Interface: [internal/usecase/boundary/auth/authenticator.go](../../internal/usecase/boundary/auth/authenticator.go)  
    - Local/test implementation: [internal/infrastructure/auth/local/auth_local.go](../../internal/infrastructure/auth/local/auth_local.go)
- Account lockout
- Data export / data deletion (user rights handling)
- Caching layer
  - A dedicated cache abstraction was considered and deliberately rejected
  - A generic `Cache` interface degrades to a lowest-common-denominator
    (a TTL-backed map) that leaks implementation semantics and discards
    technology-specific capabilities (e.g. Redis pipelines / Lua / pub-sub)
  - When caching is needed, implement it as a decorator that satisfies the
    existing domain Repository interface, so domain and usecase stay unaware
    of it — the Repository interface already provides the swap seam, so no
    new abstraction is required
