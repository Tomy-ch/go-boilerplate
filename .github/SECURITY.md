# Security Policy

This repository is a boilerplate template. This file is the initial policy provided as part of
the template; **downstream inheritors should adjust the contact points, supported versions, and
other environment-specific details to match their own setup** (environment-specific spots are
marked `※ adjust for your environment`).

A Japanese translation of this document is available at [SECURITY.ja.md](SECURITY.ja.md).

## Reporting a Vulnerability

> [!IMPORTANT]
> The contact points below are placeholders. Inheritors must replace them with their actual
> reporting channels.

- **Do not report vulnerabilities via public Issues or Pull Requests** (doing so discloses the
  information publicly).
- Report **privately** through one of the following:
  - GitHub Private Vulnerability Reporting (repository `Security` → `Advisories` → `Report a
    vulnerability`)
  - Security contact: `security@example.com` ※ adjust for your environment
- Include the following in your report:
  - Reproduction steps / PoC
  - Impact scope (expected damage, preconditions)
  - Affected version or commit SHA
- Target for the initial response: **within X business days** ※ adjust for your operations

### Supported Versions

- Latest release: ✅ supported
- Anything older: ❌ not supported ※ adjust for your operations

## Verifying release artifacts

`.github/workflows/deploy-app.yaml` attaches the following to the `runtime` image pushed to
GHCR, **targeting the digest of the pushed image**:

- cosign keyless signature (OIDC → Fulcio → Rekor)
- SLSA provenance attestation (`actions/attest-build-provenance`)
- SBOM (SPDX) attestation (`actions/attest-sbom`)

Tags are mutable but a digest is immutable, so **always verify against the digest**.
Replace `<owner>` / `<repo>` / `<tag>` / `<digest>` in the commands below to match your
environment (if you moved off GHCR, also reinterpret the `ghcr.io/<owner>` portion).

### 0. Resolve the target digest

Because verification is digest-based rather than tag-based, resolve the digest from the
target image's tag. Migrations run from this same image through a command override
(`docker run <image> /app/server migrate-up`), not from a separate one, so there is a
single digest to resolve.

```bash
docker buildx imagetools inspect ghcr.io/<owner>/app:<tag> --format '{{.Manifest.Digest}}'
crane digest ghcr.io/<owner>/app:<tag>
```

### 1. Verify the cosign signature

A keyless signature is verified by "which workflow signed it (the certificate identity)" and
"the OIDC issuer".

```bash
cosign verify \
  --certificate-identity-regexp "^https://github.com/<owner>/<repo>/\.github/workflows/deploy-app\.yaml@.*$" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  ghcr.io/<owner>/app@<digest>
```

### 2. Verify the provenance attestation (where and from what it was built)

```bash
gh attestation verify oci://ghcr.io/<owner>/app@<digest> \
  --repo <owner>/<repo> \
  --predicate-type https://slsa.dev/provenance/v1
```

### 3. Verify the SBOM (SPDX) attestation (proof of contents)

```bash
gh attestation verify oci://ghcr.io/<owner>/app@<digest> \
  --repo <owner>/<repo> \
  --predicate-type https://spdx.dev/Document
```

> [!NOTE]
> `gh attestation verify` consults the GitHub Attestation store, so it works even when the
> registry does not support OCI referrers (it can also verify via GitHub's records when the
> deploy side sets `push-to-registry: false`). `cosign verify`, by contrast, consults the
> signature (referrer) on the registry.

### Making verification a deploy gate (recommended)

> [!IMPORTANT]
> We recommend building verification into the CD pipeline as a **gate before deployment**
> (do not deploy images whose signature, provenance, and SBOM cannot be confirmed). This file
> only describes the verification steps.
