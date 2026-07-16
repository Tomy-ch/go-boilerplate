---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy, image]
---

# ADR-0087: A single runtime image with command override (no purpose-specific images)

## Status

accepted

## Context

Because the project ships a single multi-command binary (see
[ADR-0086](0086-single-multi-command-binary.md)), deployments require not only the HTTP
server role but also the migration role. A naive approach would produce a dedicated migration
image; however, that would duplicate the binary, inflate supply-chain scope, and require
coordinating image versions across two artifacts.

## Decision

Ship a single `runtime` Docker image. The default command is `./server serve` (HTTP server).
Any other role — most commonly `migrate-up` — is invoked by overriding the container command
at runtime:

```text
docker run <image> /app/server migrate-up
```

No separate migration image, job image, or worker image is produced or maintained.

## Consequences

### Positive Consequences

- One image digest to sign, attest, and verify per deployment. Supply-chain integrity
  checks apply uniformly to all roles (see [ADR-0091](0091-release-image-supply-chain.md)).
- The migration binary is always exactly the same version as the serving binary, eliminating
  version skew between migration and application code.
- Reduced CI surface: one build step, one push, one set of tags.

### Negative Consequences

- The image is slightly larger than a hypothetical migration-only image because all roles
  are included. In practice the binary is statically linked and the delta is small.
- Operators unfamiliar with command override may attempt to run migrations via a separate
  (possibly stale) image.

## Alternatives Considered

### Dedicated migration image

A separate Dockerfile target or image built from the same binary. Doubles the number of
images to manage and sign, and introduces the risk of deploying a migration binary that
differs from the serving binary.

### Init-container that pulls migration binary separately

The binary version must be coordinated externally. Rejected: any version skew between
migration and app binary is a latent correctness risk.

## Notes

- `docker/server/Dockerfile` lines 55-57: `CMD ["/app/server", "serve"]` is the default;
  migrations override this to `["/app/server", "migrate-up"]`.
- `docker/server/README.md`: "Schema migrations run from the `runtime` image via command
  override, so no dedicated migration target exists."
- Migration orchestration (one-shot pre-deploy job) is a separate concern; see
  ADR-0090.
