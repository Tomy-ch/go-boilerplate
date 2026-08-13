---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy]
---

# ADR-0099: Deploy is a vendor-neutral skeleton (build/sign implemented; cloud CD is a stub; registry not fixed)

## Status

accepted

## Context

The deploy pipeline must demonstrate a production-grade supply chain without binding the
repository to a specific cloud provider or container registry. The build, signing, and
attestation steps are universally applicable; the cloud-specific steps (registry login,
credential configuration, migration job invocation, and application deployment) vary per
environment and cannot be implemented in a generic way.

Implementing the pipeline fully for one cloud provider (e.g., AWS ECS) would force any
deployment to a different provider to strip those steps out, creating friction. Shipping no
pipeline at all would leave no supply-chain-integrity reference.

This tension is resolved by separating what can be implemented generically from what must
be customized (see [ADR-0001](0001-avoid-lock-in.md)).

## Decision

The deploy workflow (`deploy-app.yaml`) is structured as a **vendor-neutral skeleton**:

- **Fully implemented** (no modification needed): image build, tagging, caching
  (`docker/build-push-action`), cosign signing, build-provenance attestation, and SBOM
  attestation (see [ADR-0098](0098-release-image-supply-chain.md)).
- **Stubs** (must be replaced for the target environment):
  - `Define registry` / `Login to registry` — registry selection and authentication.
    The `meta_registry` output variable is referenced by all downstream steps so a single
    replacement propagates everywhere. Currently defaults to `ghcr.io` as an illustrative
    example.
  - `Configure cloud credentials` — OIDC / Workload Identity / Federated Identity setup.
  - `Run migration (one-time job)` — cloud-native job primitive (ECS RunTask, Cloud Run
    Job, Kubernetes Job, etc.).
  - `Deploy application` — cloud-native deploy primitive (ECS service update, Cloud Run
    deploy, Kubernetes rollout, etc.).

Each stub contains explanatory `echo` lines that describe the expected behavior, the
environment constraint, and provider examples, so what to implement is documented without
forcing any one provider.

## Consequences

### Positive Consequences

- Any cloud (AWS / GCP / Azure / on-premises) can use the workflow without removing
  cloud-specific steps for a rival provider.
- The registry is not hard-coded: replacing the two registry steps is sufficient to switch
  registries; signing and attestation steps follow automatically via `meta_registry`.
- Supply-chain integrity (signing, provenance, SBOM) is pre-wired and works out of the box
  regardless of which cloud is targeted.

### Negative Consequences

- The pipeline is not runnable end-to-end until the stub steps are implemented for the
  target environment. A pipeline that partially executes (build succeeds, deploy stub does
  nothing) can give a false sense of completeness.
- Stubs must be kept accurate and up to date as the surrounding pipeline evolves, or they
  risk misleading whoever implements them.

## Alternatives Considered

### Fully implemented for one cloud (e.g., AWS ECS)

Provides a working example for one target environment, but forces every other one to
understand and remove AWS-specific steps. Inconsistent with [ADR-0001](0001-avoid-lock-in.md).

### External CD tool (Argo CD, Flux, Spinnaker)

Moves the deploy concern out of GitHub Actions entirely. Valid for mature platforms but
introduces a new tool dependency that exceeds this repository's deploy scope and is itself
provider-specific in its hosting.

## Notes

- `.github/workflows/deploy-app.yaml` `Define registry` step and `Login to registry` step
  are the registry customization points.
- `.github/workflows/deploy-app.yaml` (`Configure cloud credentials`, `Run
  migration`, `Deploy application`) are the cloud-CD stubs.
- The lock-in avoidance principle is ADR-0001.
- Source: `.github/workflows/deploy-app.yaml`.
