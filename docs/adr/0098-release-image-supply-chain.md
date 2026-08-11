---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy, security, supply-chain]
---

# ADR-0098: Release-image supply-chain integrity (cosign signing + provenance + SBOM)

## Status

accepted

## Context

Container images that are pushed to a registry and deployed to production are part of the
software supply chain. Without integrity guarantees, there is no cryptographic proof that
the image running in production was built from the expected source commit by the expected
CI pipeline, and no inventory of what software is inside the image.

Supply-chain attacks on container registries (image tag mutation, registry credential
compromise) and build systems make these guarantees increasingly important. Industry
standards (SLSA, SSDF) recommend provenance attestation and software bills of materials
(SBOM) as baseline controls.

## Decision

Every release image pushed to the registry receives three supply-chain integrity artifacts,
all applied to the immutable image digest rather than a mutable tag:

1. **cosign keyless signing** — `cosign sign --yes` signs the image digest using Sigstore's
   keyless flow (OIDC identity from the GitHub Actions workflow). No long-lived key material
   is required.

2. **Build provenance attestation** — `actions/attest-build-provenance` records a SLSA
   provenance statement (source commit, workflow run, actor) and attaches it to the image
   digest. The attestation is stored in the GitHub Attestation store and optionally pushed
   to the registry as an OCI 1.1 referrer.

3. **SBOM generation and attestation** — `anchore/sbom-action` generates an SPDX-JSON SBOM
   from the pushed image, then `actions/attest-sbom` attaches it to the image digest as a
   second attestation. This SBOM is distinct from any scan-time SBOM produced by
   `image-scan.yaml`; both serve different purposes and must not be removed.

All three artifacts target the image digest produced by the `Build and push image` step,
ensuring they are bound to an immutable reference.

Registry portability note: the `meta_registry` step and the `Login to registry` step are
explicitly marked as setup-team customization points. The signing and attestation steps
reference `meta_registry` rather than a hard-coded registry, so they follow the registry
choice without modification.

## Consequences

### Positive Consequences

- Consumers can verify the image was built by this workflow from the declared commit using
  `cosign verify` and `gh attestation verify`.
- Provenance and SBOM are stored as first-class OCI artifacts alongside the image, making
  them discoverable through standard OCI tooling (when the registry supports OCI 1.1
  referrers).
- No key management overhead: the keyless flow uses the GitHub Actions OIDC token.

### Negative Consequences

- Registries that do not support OCI 1.1 referrers cannot store the attestation as a
  referrer; the workflow comment (`push-to-registry: true`) must be changed to `false` for
  such registries, falling back to GitHub Attestation store only.
- The signing and attestation steps add to build job duration.
- cosign and the attestation actions are pinned by commit digest; updating them requires
  a deliberate change to the workflow file.

## Alternatives Considered

### Long-lived signing key (cosign with a stored key)

Requires secret rotation, key storage, and access control. The keyless flow provides
equivalent verification with lower operational overhead in a GitHub-hosted CI context.

### No supply-chain artifacts

Simpler pipeline, but no way to verify image provenance or contents after the fact.
Rejected as inconsistent with the lock-in avoidance and operability goals
([ADR-0001](0001-avoid-lock-in.md)).

## Notes

- `.github/workflows/deploy-app.yaml` implements the three integrity steps
  (`Attest build provenance` / `Sign image (cosign keyless)` / `Attest SBOM`).
- The SBOM generated here is the post-push, attestation-attached SBOM; the scan-time SBOM
  in `image-scan.yaml` serves a different purpose (vulnerability scanning) and must coexist.
- Source: `.github/workflows/deploy-app.yaml`.
