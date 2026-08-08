# Architecture Decision Records (ADR)

This directory holds the project's **architecture decisions**, one immutable record per
file, in [MADR-lite](https://adr.github.io/madr/) form. It supersedes the former monolithic
`docs/decisions.md` (now a redirect stub).

An ADR captures a single decision at a point in time: the context, the options weighed,
the choice, and its consequences. Superseding a decision does **not** mean editing its
ADR — it means adding a *new* ADR whose `Status` is `accepted` and marking the old one
`superseded`. The record of *why we once chose X* is preserved.

## What belongs here (and what does not)

| Kind | Example | Home |
| --- | --- | --- |
| **decision** — a choice among alternatives with lasting consequences | "Adopt onion architecture" | this dir (ADR) |
| **exclusion** — a deliberate "we intentionally do NOT do X" | "No in-application rate limiter" | this dir (ADR) |
| **rule** — a day-to-day enforced constraint / consequence of a decision | "Controller must not import infrastructure" | `docs/rules.md` (may link an ADR) |
| **inventory** — a catalog that drifts with the code | the direct-dependency table | `docs/reference/dependencies.md` (living doc) |

The dependency inventory is **not** an ADR: it tracks `go.mod` and changes continuously,
which is the opposite of an immutable record. The *policy* for selecting dependencies is
a decision (an ADR); the *list* of them is a living reference.

## Conventions

- **Filename**: `NNNN-kebab-title.md`, zero-padded 4 digits. A number freed by supersession is never re-assigned to a different decision, and neither is a number freed by the consolidation pass below — after a consolidation the range is contiguous again, but no surviving ADR inherits a retired ADR's old number.
- **Ordering**: numbers follow dependency / foundational order (principles → contract → layers → subsystems → cross-cutting → exclusions), not discovery order. To preserve this order, a new ADR may be **inserted at its thematic position by shifting all subsequent numbers +1** (a pure renumbering: every shifted ADR keeps its content, and all repository-internal references are updated in the same change). External references to ADR numbers from before such a shift may be stale — the kebab title in the filename is the stable identifier.
- **Status lifecycle**: `proposed` → `accepted` → (`superseded` | `deprecated`).
- **Amendment**: an `accepted` ADR is amended **in place** — update `date`, keep `status: accepted`, and do not create a superseding ADR for what is still the same decision. This repository is a boilerplate: what it ships is the current design, not the sequence of positions that produced it, and a fork that must read three ADRs to learn one rule pays for history it did not live. When an amendment changes the conclusion, the position it replaces moves to Alternatives Considered with the reason it was dropped — nothing is discarded, it changes section. The insertion renumbering above changes only the number, not the record.
- **Who may amend**: amendment is a decision for this repository's architect or tech lead, taken per amendment. Finding that an accepted ADR is contradicted by the implementation is a reason to raise it with them, not a licence to amend it — the implementation is as likely to be the error. A previous amendment is not standing authorization for the next.
- **A new ADR** is for a decision that should be read independently of the one beside it, not for a revision of one. `superseded` remains in the lifecycle for a decision genuinely replaced rather than revised.
- **Consolidation exception (authorised, one-off per harvest)**: this repository is a boilerplate whose sample feature set is developed, harvested, and then removed. Implementing a sample produces ADRs that are part architectural decision and part feature detail, and they accumulate at the tail of the numbering in discovery order — which is exactly what the ordering convention above exists to prevent. A **consolidation pass may therefore merge, rewrite, and retire such ADRs**, feeding the architectural residue back into the ordered set and moving the feature content to `docs/spec/`. This is a deliberate exception to immutability, and it is bounded: it applies only to ADRs produced by sample development, it is performed as one reviewed change, and every retired ADR's architectural content survives in the ADR that absorbed it — nothing is discarded, only relocated. Outside a consolidation pass, an ADR that is still the same decision is amended in place per the Amendment convention above; what this exception adds is the authority to merge and retire files, which an amendment does not have.
- **Template**: copy [`template.md`](template.md).
- **Meta**: [`0000-record-architecture-decisions.md`](0000-record-architecture-decisions.md) records the decision to use ADRs and this classification.
- **Translation**: each ADR mirrors to `docs/ja/adr/` (via the `canonicalize-doc` flow).
- **Exclusion ADRs** (deliberate "we do NOT do X") carry a `setup-review` tag so the repository-setup flow can enumerate them. At initial setup a fork may **edit these directly** to establish its own baseline; the supersede-by-new-ADR model applies only to changes made later. See `docs/get-started/setup-repository.md` Phase 12.

## Log

All decisions from `docs/decisions.md` and the latent decisions across the repository have
been materialized as ADRs. Numbering follows dependency / foundational order (principles →
contract → HTTP → persistence → DI/config → async subsystems → observability →
toolchain/CI → process → binary/deploy → exclusions). Exclusion ADRs (deliberate "we do NOT
do X") are tagged `setup-review`.

| # | Decision | Status |
| --- | --- | --- |
| [0000](0000-record-architecture-decisions.md) | Record architecture decisions as ADRs | accepted |
| [0001](0001-avoid-lock-in.md) | Adopt lock-in avoidance as a design principle | accepted |
| [0002](0002-onion-architecture.md) | Adopt pragmatic onion architecture | accepted |
| [0003](0003-interface-based-decoupling.md) | Define boundaries with interfaces for loose coupling (DIP) | accepted |
| [0004](0004-modular-monolith.md) | Adopt a modular monolith (microservices are a non-goal) | accepted |
| [0005](0005-driving-adapters-not-split-axis.md) | REST / Worker / Job are driving adapters, not a service-split axis | accepted |
| [0006](0006-structural-safety-via-tooling.md) | Enforce structural safety with tooling and CI (depguard) | accepted |
| [0007](0007-agents-md-operational-contract.md) | With-AI development — AGENTS.md as the operational contract | accepted |
| [0008](0008-docs-as-canonical-source.md) | Docs-as-canonical-source strategy (English canonical + ja mirror + portal) | accepted |
| [0009](0009-openapi-first.md) | Define the API contract OpenAPI-first | accepted |
| [0010](0010-redocly-modular-spec-pipeline.md) | Author the spec in modular Redocly files, bundle, then generate | accepted |
| [0011](0011-oapi-codegen-strict-server.md) | Generate per tag/handler with oapi-codegen in strict-server mode | accepted |
| [0012](0012-retain-generated-openapi.md) | Retain the bundled openapi.gen.yaml as a committed cross-repo contract artifact | accepted |
| [0013](0013-spec-driven-request-validation.md) | Validate requests and enforce auth from the spec at runtime; do not validate responses | accepted |
| [0014](0014-validation-value-authority.md) | Designate the domain layer as the sole authority for business-validity rules | accepted |
| [0015](0015-boundary-value-ownership.md) | OpenAPI is the wire contract, not the domain rule; request is subset of domain, domain is subset of response | accepted |
| [0016](0016-metrics-endpoint-auth-exception.md) | /metrics is an auth exception — outside OpenAPI validation, protected by a separate BasicAuth middleware | accepted |
| [0017](0017-echo-http-framework.md) | Adopt Echo as the HTTP framework | accepted |
| [0018](0018-priority-ordered-middleware-chain.md) | Build the middleware chain as a priority-ordered, data-driven list | accepted |
| [0019](0019-outbound-http-resilience.md) | Provide an outbound-HTTP resilience foundation (retry / circuit breaker / retry budget / dual timeout) | accepted |
| [0020](0020-egress-ssrf-guard.md) | Adopt an egress SSRF / dial-guard security posture for outbound HTTP | accepted |
| [0021](0021-sql-first-data-access.md) | SQL-first data access | accepted |
| [0022](0022-sqlc-type-safe-sql.md) | Generate type-safe SQL access with sqlc | accepted |
| [0023](0023-merged-dml-schema-as-sqlc-input.md) | Use merged DML and a dumped schema as sqlc's single input | accepted |
| [0024](0024-append-only-immutable-migrations.md) | Treat migrations as append-only and immutable | accepted |
| [0025](0025-sequential-migration-ids.md) | Use sequential 6-digit migration IDs with CI-enforced gap and pair checks | accepted |
| [0026](0026-master-data-via-migration.md) | Ship master data via migration; keep transactional seed out of production | accepted |
| [0027](0027-list-query-access-path-index.md) | Index a filtered, ordered list query along the access path the planner actually takes | accepted |
| [0028](0028-lightweight-cqrs.md) | Adopt lightweight CQRS — Repository for writes, QueryService for reads | accepted |
| [0029](0029-system-cqrs-dml-category.md) | Introduce system_cqrs as a fourth DML category outside the CQRS split | accepted |
| [0030](0030-commandservice-atomicity-criterion.md) | Cross-aggregate operations: usecase + outbox by default, a synchronous lock when a guard must not go stale, CommandService only for single-tx atomicity | accepted |
| [0031](0031-transaction-retry-idempotent-callers.md) | Retry transactions on serialization conflict; require callers to be idempotent | accepted |
| [0032](0032-ordered-pessimistic-row-locks.md) | Serialize contended writes with ordered pessimistic row locks taken before the guarded condition | accepted |
| [0033](0033-uuidv7-identifiers.md) | Use UUIDv7 (time-ordered) identifiers for all entity primary keys | accepted |
| [0034](0034-two-scale-quantity-model.md) | Hold a quantity in two scales — exact decimal for precision, integer minor unit for settlement | accepted |
| [0035](0035-domain-lexicon.md) | Cross-aggregate value objects live in a curated domain lexicon (`internal/domain/lexicon`) | accepted |
| [0036](0036-uber-fx-di.md) | Adopt Uber Fx for dependency injection and lifecycle | accepted |
| [0037](0037-fx-neutral-di-abstraction.md) | Contain fx behind a neutral DI abstraction (Registrar / Shutdowner) | accepted |
| [0038](0038-env-gated-wiring.md) | Swap implementations per environment via DI (env-gated wiring) | accepted |
| [0039](0039-subsystem-typed-config-loaders.md) | Subsystem-scoped envPrefix typed config loaders | accepted |
| [0040](0040-config-default-vs-required-governance.md) | Governance: default-in-code (immutable) vs required-in-file (variable) | accepted |
| [0041](0041-immutable-fail-fast-config.md) | Config is immutable, loaded once at startup, fail-fast | accepted |
| [0042](0042-embedded-self-contained-binary.md) | go:embed bundles config (.env) and migrations for a self-contained binary | accepted |
| [0043](0043-apperror-protocol-agnostic-errors.md) | Protocol-agnostic aggregated error classification (apperror) | accepted |
| [0044](0044-error-metadata-code-message-details.md) | Protocol-neutral error metadata (code / message / details) on top of apperror | accepted |
| [0045](0045-error-details-opt-in-gate.md) | Opt-in gate for error-response details via schema split (refines 0043) | accepted |
| [0046](0046-broker-agnostic-worker-scaffold.md) | Broker-agnostic pull-ack worker scaffold | accepted |
| [0047](0047-out-of-scope-push-streaming-brokers.md) | Push-type brokers and streaming-log platforms are out of scope for the worker port | accepted (exclusion) |
| [0048](0048-sqs-adapter-opt-in.md) | SQS adapter is opt-in and not linked into the default binary | superseded by [0049](0049-broker-sdk-isolation-verified-after-sample-removal.md) |
| [0049](0049-broker-sdk-isolation-verified-after-sample-removal.md) | Broker-SDK isolation is verified after sample removal, not by leaving the adapter unwired | accepted |
| [0050](0050-transactional-outbox.md) | Transactional outbox: emit events within the business transaction | accepted |
| [0051](0051-at-least-once-outbox-poll.md) | At-least-once delivery via polling (transport-level retry disabled) | accepted |
| [0052](0052-skip-locked-outbox-relay.md) | Single-transaction relay using SELECT FOR UPDATE SKIP LOCKED (safe across instances) | accepted |
| [0053](0053-message-id-idempotency-propagation.md) | Propagate the outbox message_id as the receiver's Idempotency-Key | accepted |
| [0054](0054-outbox-dead-after-max-attempts.md) | MaxAttempts = 10, then the message is dead (terminal until manual replay) | accepted |
| [0055](0055-outbox-retention-gc.md) | 7-day retention GC of published rows (batches of 10,000) | accepted |
| [0056](0056-publisher-http-profile-isolation.md) | Isolate the publisher's non-standard HTTP profile inside the relay | accepted |
| [0057](0057-relay-resident-gc-oneshot.md) | The relay is a resident process; GC is a one-shot cron job | accepted |
| [0058](0058-single-tx-at-most-once-idempotency.md) | Run claim, business function, and complete in a single transaction for at-most-once semantics | accepted |
| [0059](0059-idempotency-scope-required.md) | Every Store call requires an explicit scope to prevent cross-user key collisions | accepted |
| [0060](0060-idempotency-fixed-ttl.md) | Fix idempotency key TTL at 24 hours with no per-route configuration | accepted |
| [0061](0061-idempotency-response-persistence.md) | Persist the response body as JSON to enable deterministic replay (accepted PII tradeoff) | accepted |
| [0062](0062-idempotency-gc-separate-job.md) | Run idempotency key garbage collection as a separate one-shot CLI job | accepted |
| [0063](0063-idempotency-orthogonal-concerns.md) | Keep idempotency orthogonal to optimistic locking and rate limiting | accepted (exclusion) |
| [0064](0064-job-fresh-fx-app-per-run.md) | Each job launch constructs a fresh fx.App (one-shot lifecycle) | accepted |
| [0065](0065-job-no-worker-machinery.md) | Jobs deliberately have no broker, circuit breaker, drain, or health machinery | accepted (exclusion) |
| [0066](0066-job-explicit-registration.md) | Jobs are explicitly registered (no auto-discovery) | accepted |
| [0067](0067-config-driven-observability-gating.md) | Config-driven observability gating | accepted |
| [0068](0068-vendor-neutral-otlp-export.md) | Vendor-neutral OTLP-only export (delegate backend to the Collector) | accepted |
| [0069](0069-official-otel-semconv.md) | Use only official OpenTelemetry semantic conventions; do not invent custom semconv or put vendor keys in typed config | accepted (exclusion) |
| [0070](0070-dual-path-metrics.md) | Metrics travel two paths — OTLP push and Prometheus scrape | accepted |
| [0071](0071-lifecycle-independent-provider.md) | Observability providers are lifecycle-independent (ProviderShutdowner) | accepted |
| [0072](0072-fixed-default-sampling.md) | Fix the SDK default sampling; do not expose sampling as an env knob | accepted (exclusion) |
| [0073](0073-library-selection-policy.md) | Single-responsibility library selection policy | accepted |
| [0074](0074-bridge-instrumentation-exceptions.md) | Bridge / instrumentation libraries as bounded SRP exceptions | accepted |
| [0075](0075-containerized-pinned-toolchain.md) | Use a containerized toolchain pinned by mise for reproducibility | accepted |
| [0076](0076-mise-ssot-drift-gate.md) | mise.toml is the single source of truth; versions propagate downstream with a CI drift gate | accepted |
| [0077](0077-make-single-entrypoint.md) | Make is the single tool entrypoint with .mk registration and self-documenting help | accepted |
| [0078](0078-scripts-in-node-go.md) | Operational scripts live in scripts/ as Node (.mjs) or Go; shell scripting is not used | accepted |
| [0079](0079-docker-compose-dev-environment.md) | Local dev environment is provided via Docker Compose with profile-separated services | accepted |
| [0080](0080-two-layer-golangci-config.md) | Two-layer golangci config: minimal default vs full authoritative gate | accepted |
| [0081](0081-local-hooks-mirror-ci.md) | Local git hooks duplicate the CI contract (local == CI, glob-scoped, bypass-then-verify-once) | accepted |
| [0082](0082-coverage-hard-gate.md) | Total coverage 90% is a CI hard gate, with an exception-governance path | accepted |
| [0083](0083-ci-real-graph-boot-check.md) | CI boots the real fx graph against real Postgres (startup verification) | accepted |
| [0084](0084-generated-artifact-drift-gate.md) | Generated-artifact drift gate + release-branch-centralized auto-generation bot | accepted |
| [0085](0085-multi-layer-security-scanning.md) | Multi-layer security scanning, splitting reporting from gating, on hardened runners | accepted |
| [0086](0086-sha-pinned-actions.md) | Pin GitHub Actions by SHA with a supply-chain quarantine | accepted |
| [0087](0087-malicious-package-detection-via-cooldown.md) | Malicious packages are mitigated by a publication cooldown, with no dedicated detector adopted | accepted |
| [0088](0088-rollback-integration-tests.md) | Run infrastructure integration tests against a real DB with sentinel-error rollback | accepted |
| [0089](0089-multi-model-adversarial-review.md) | Use multi-model adversarial review with finder and verifier subagents | accepted |
| [0090](0090-lean-a-spec-scaffold.md) | Scaffold only domain and usecase from spec files; derive controller and infra from generated code | accepted |
| [0091](0091-cli-humble-object-split.md) | CLI humble-object split (thin cmd/ shell + testable internal/cli core) | accepted |
| [0092](0092-single-multi-command-binary.md) | All roles are one multi-command binary | accepted |
| [0093](0093-single-runtime-image.md) | A single runtime image with command override (no purpose-specific images) | accepted |
| [0094](0094-hardened-alpine-runtime.md) | Use a hardened-alpine runtime base; do NOT use distroless/scratch | accepted (exclusion) |
| [0095](0095-per-environment-images.md) | Per-environment images (.env matrix x APP_ENV build-arg, fixed at build time) | accepted |
| [0096](0096-predeploy-oneshot-migration.md) | Migrations run as a pre-deploy one-shot; do NOT auto-migrate at application startup | accepted (exclusion) |
| [0097](0097-release-image-supply-chain.md) | Release-image supply-chain integrity (cosign signing + provenance + SBOM) | accepted |
| [0098](0098-vendor-neutral-deploy-skeleton.md) | Deploy is a vendor-neutral skeleton (build/sign implemented; cloud CD is a template; registry not fixed) | accepted |
| [0099](0099-docs-via-github-pages.md) | Publish static docs/ via GitHub Pages (released on production push) | accepted |
| [0100](0100-no-in-app-rate-limiter.md) | Do not provide an in-application rate limiter | accepted (exclusion) |
| [0101](0101-scheduled-job-concurrency-delegated.md) | Do not control scheduled-job concurrency in-app; delegate to the scheduler | accepted (exclusion) |
| [0102](0102-no-generic-cache-abstraction.md) | Do not provide a generic Cache abstraction | accepted (exclusion) |
| [0103](0103-outbox-relay-hardening-delegated.md) | Delegate outbox-relay duplicate-window hardening (multi-layer lease redesign) to production copies | accepted (exclusion) |

Frontmatter fields: `status`, `date`, `deciders`, `supersedes` / `superseded-by`, `tags`.
Consequences follow the MADR standard (`Positive` / `Negative`; optional `Neutral`).
