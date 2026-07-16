---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy]
---

# ADR-0091: Deploy is a vendor-neutral skeleton (build/sign implemented; cloud CD is a template; registry not fixed)

## Status

accepted

## Context

The scaffold must demonstrate a production-grade CI/CD pipeline without locking adopters
into a specific cloud provider or container registry. The build, signing, and attestation
steps are universally applicable; the cloud-specific steps (registry login, credential
configuration, migration job invocation, and application deployment) vary per environment
and cannot be implemented in a generic way.

Shipping a fully implemented pipeline for one cloud provider (e.g., AWS ECS) would force
adopters on other clouds to strip out those steps, creating friction. Shipping no pipeline
at all would leave adopters without a supply-chain-integrity reference.

This tension is resolved by separating what can be implemented generically from what must
be customized (see [ADR-0001](0001-avoid-lock-in.md)).

## Decision

The deploy workflow (`deploy-app.yaml`) is structured as a **vendor-neutral skeleton**:

- **Fully implemented** (no modification needed): image build, tagging, caching
  (`docker/build-push-action`), cosign signing, build-provenance attestation, and SBOM
  attestation (see [ADR-0090](0090-release-image-supply-chain.md)).
- **Template stubs** (must be replaced by setup teams):
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
environment constraint, and provider examples, so adopters understand what to implement
without being forced toward any one provider.

## Consequences

### Positive Consequences

- Adopters on any cloud (AWS / GCP / Azure / on-premises) can use the workflow without
  removing cloud-specific steps from a rival provider.
- The registry is not hard-coded: replacing the two registry steps is sufficient to switch
  registries; signing and attestation steps follow automatically via `meta_registry`.
- Supply-chain integrity (signing, provenance, SBOM) is pre-wired and works out of the box
  regardless of which cloud the adopter targets.

### Negative Consequences

- The pipeline is not runnable end-to-end without setup-team customization of the stub
  steps. A pipeline that partially executes (build succeeds, deploy stub does nothing)
  can give a false sense of completeness.
- Template stubs must be kept accurate and up to date as the surrounding pipeline evolves,
  or they risk misleading adopters.

## Alternatives Considered

### Fully implemented for one cloud (e.g., AWS ECS)

Provides a working example for one adopter segment, but forces every other adopter to
understand and remove AWS-specific steps. Inconsistent with [ADR-0001](0001-avoid-lock-in.md).

### External CD tool (Argo CD, Flux, Spinnaker)

Moves the deploy concern out of GitHub Actions entirely. Valid for mature platforms but
introduces a new tool dependency that exceeds the scaffold's scope and is itself
provider-specific in its hosting.

## Notes

- `.github/workflows/deploy-app.yaml` `Define registry` step (lines 67-72) and `Login to
  registry` step (lines 96-102) are the registry customization points.
- `.github/workflows/deploy-app.yaml` lines 181-218 (`Configure cloud credentials`, `Run
  migration`, `Deploy application`) are the cloud-CD template stubs.
- The lock-in avoidance principle is ADR-0001.
- Source: `.github/workflows/deploy-app.yaml`.
