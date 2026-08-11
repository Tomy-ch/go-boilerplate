# Roadmap

English | [日本語](../ja/project/roadmap.ja.md)

<!-- boilerplate-only:replace-begin -->
This page records the **direction** this project is being maintained in — the standing commitments
that shape what gets accepted, not a schedule. Individual work items live in the issue tracker,
where they can be closed; a list of them here would be a second copy that goes stale the moment one
of them lands.
<!-- boilerplate-only:replace-with -->
<!-- = This page records the **direction** this project is being maintained in — the standing -->
<!-- = commitments that shape what gets accepted, not a schedule. Individual work items belong in -->
<!-- = the issue tracker, where they can be closed. -->
<!-- = -->
<!-- = Replace the sections below with your project's own direction. What is inherited from the -->
<!-- = template is the shape of the page, not its content. -->
<!-- boilerplate-only:replace-end -->

Nothing here is a commitment to a date. What the maintainer does and does not undertake is stated
in [policy.md](policy.md), and it is deliberately narrower than a roadmap normally implies.

<!-- boilerplate-only:begin -->
## What each release line was about

A release line here is a **subject**, not a batch of features: one question the template was trying
to answer, worked until it was answered. Reading them in order is the fastest way to see why the
repository is shaped the way it is — and the per-release detail stays in the release notes rather
than being copied here.

### v1 — becoming a template you can grow, not a sample you read once

The v1 line took a working API server and made it something a team could adopt: the layering stated
as rules rather than implied by the code, the contracts generated rather than hand-kept, and the
setup and release operations owned by the repository itself.

- **[v1.0.0](https://github.com/Tomy-ch/go-boilerplate/releases/tag/v1.0.0)** established the base —
  the canonical documents (`architecture` / `rules` / `development-flow`), the agent contract, the
  move to `pgx/v5`, the documentation portal, the setup helpers, and release-centric branching.
- **[v1.1.0](https://github.com/Tomy-ch/go-boilerplate/releases/tag/v1.1.0)** answered the first
  round of adoption feedback: hardened setup scripts, and the English / Japanese README pairing that
  the repository has kept since.
- **[v1.2.0](https://github.com/Tomy-ch/go-boilerplate/releases/tag/v1.2.0)** ran one feature through
  every layer at once (the user detail / update / soft-delete operations), pinned the toolchain to a
  single source of truth, and added the first security and generated-artifact-drift gates.
- **[v1.3.0](https://github.com/Tomy-ch/go-boilerplate/releases/tag/v1.3.0)** deliberately added
  almost nothing: a repository-wide quality review, with the findings applied back into the
  implementation, the comments, and the error classification.
- **[v1.4.0](https://github.com/Tomy-ch/go-boilerplate/releases/tag/v1.4.0)** closed the line by
  making it learnable and removable — a tutorial that builds one feature from nothing, a
  multi-level documentation portal, a supply-chain layer in CI, and the tool that deletes the
  sample API.

### v2 — running as a backend, not as an API server

The v2 line is about everything a service needs that a request / response cycle does not show:
work that happens after the response, state that survives a retry, and the ability to see what
happened. Its second theme is **self-containment** — the stack a developer starts should be the
whole system, not the parts that happen to run without a cloud account.

- **[v2.0.0](https://github.com/Tomy-ch/go-boilerplate/releases/tag/v2.0.0)** added the
  quasi-distributed foundation: the worker scaffold, the transactional outbox, idempotent request
  handling, authorization, resilient outbound HTTP, and OpenTelemetry with a local collector stack
  to receive it.
- **[v2.1.0](https://github.com/Tomy-ch/go-boilerplate/releases/tag/v2.1.0)** hardened exactly those
  subsystems rather than extending them, and raised the quality of the API contract itself — error
  metadata with opt-in detail exposure, domain inputs as value objects with collect-all validation,
  and SSRF / idempotency / relay hardening.
- **The current line** completes the self-contained stack by emulating the remaining external
  dependencies locally — S3-compatible object storage, an SQS-compatible broker, and a development
  OIDC provider — alongside the supply-chain controls and the fork tooling that a template needs in
  order to be forked safely.

Development with AI assistance runs across both lines rather than belonging to either: it entered in
v1.2.0 as skills, was parallelized in v1.4.0, and is now an agent contract plus the mechanical gates
that check what an agent produced.

### v3 — staying a modular monolith while being ready to distribute

The next line's subject is support for genuine distribution: **a boilerplate that keeps its modular
monolith and behaves as a distributed system the moment a boundary is crossed.** Becoming
microservices is not the premise — the point is that inter-module calls go through a contract, so
distributing one boundary becomes a swap of the adapter behind it rather than a redesign.

The requirements that follow from that, and how they are prioritized, are in
[distributed-ready-architecture.md](../plan/distributed-ready-architecture.md).

<!-- boilerplate-only:end -->
## Standing directions

**Runtimes and toolchains follow their stable lines, on the upstream's clock.** A new major is
adopted once it reaches the support status the project depends on — for a language runtime, its LTS
designation — rather than when it is announced. The consequence is that some upgrades sit visibly
pending for months with a known date attached; that is the policy working, not a backlog.
The procedure for the Go runtime is [go-upgrade.md](../maintenance/go-upgrade.md).

**Supply-chain controls deepen; they do not loosen.** Cooldown windows before a fresh release can be
adopted, pinning by digest rather than by tag, and attestations over released artifacts are treated
as the floor. A change that removes one of them needs to argue against the threat model in
[security.md](../design/security.md), not merely against the inconvenience.

**Mechanically decidable rules keep moving into tooling.** Where a convention is currently enforced
by review, the preferred direction is to make it a lint rule, an architecture test, or a
generation-drift check — so the rule is enforced identically for every contributor and every agent,
and review spends itself on the judgements that cannot be automated.

**Deliberate exclusions stay excluded.** The list in [out-of-scope.md](out-of-scope.md) is not a
backlog. Items move off it only when the reason recorded there stops holding, and that is an ADR-level
decision.

<!-- boilerplate-only:begin -->
## Microservices are permanently out of scope

What v3 provides is the ability to distribute a boundary once it needs to be distributed — not a
microservice platform. That gap is not scheduled to close, and there is no intention to close it.

- **The unit being optimized for differs.** A microservice template optimizes for the minimum shape
  of one service: one bounded context, thin enough to deploy alone. This repository optimizes for
  holding several boundaries inside one deployable while enforcing the boundaries between them. v3
  makes the move to the former possible, and **making something possible is not the same as being
  the best fit for it**.
- **The hard part sits outside the application.** An orchestration platform, a distributed tracing
  backend, service discovery and traffic control, per-service CI and deployment. Kubernetes is the
  common instance of this rather than a requirement; the point is that none of it is territory this
  repository owns. Deployment implementation, IaC, and observability operations are already listed
  in [out-of-scope.md](out-of-scope.md).
- **A template should not pre-empt that decision.** By the time microservices are genuinely needed,
  the constraint that produced the need — organizational structure, regulation, fault isolation — is
  concrete, and the platform is chosen against that constraint. A template that bakes in one answer
  first ends the choice before anyone has looked at the constraint.

So Kubernetes manifests, service-mesh-dependent communication, and per-service deployment pipelines
will not arrive here. What this line defends instead is that **choosing not to distribute stays valid
all the way through**.

**Out of scope is not forbidden.** It means this repository does not supply it, not that you may not
reshape what is here into a microservice architecture. The obligation on this side is to keep it
**traceable which premises you would be rewriting** if you decided to. Every deliberate exclusion is
recorded as an ADR, and a fork rewrites those directly at setup time to establish its own baseline
(Phase 13 of [setup-repository.md](../get-started/setup-repository.md)).

## Beyond this repository

Companion boilerplates are planned along the same lines — frontend, infrastructure, and
observability — so that the boundaries this repository assumes on its edges have a counterpart that
assumes the same ones. They are separate repositories rather than additions here: nothing in this
project's scope changes when they land.

A microservice platform, if it is ever taken on, takes the same form: one product that integrates
the orchestration platform, the observability platform and the application, built separately rather
than bolted on here — because, as above, the difficulty lives in that integration and not on the
application side.

<!-- boilerplate-only:end -->
## Where the actual work is

Planned and in-progress work lives in the issue tracker. Anything that changes *why* the system is
shaped the way it is is additionally recorded as an ADR under [adr/](../adr/README.md) — read in
sequence, those records are the honest history of the project's direction, and this page is only its
current summary.
