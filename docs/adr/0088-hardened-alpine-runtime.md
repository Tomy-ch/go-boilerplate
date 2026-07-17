---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy, image, exclusion, setup-review]
---

# ADR-0088: Use a hardened-alpine runtime base; do NOT use distroless/scratch

## Status

accepted

## Context

Minimal runtime base images (distroless, scratch) are often proposed for Go services because
Go compiles to a static binary and technically needs no OS layer at runtime. The appeal is a
smaller attack surface and a smaller image.

However, the runtime image here must satisfy three concrete requirements:

1. **TLS trust** — outbound HTTPS calls require a trusted CA certificate bundle.
2. **Timezone data** — time-zone-aware business logic (`tzdata`) must resolve correctly.
3. **Security patching** — the base layer must be independently patchable via the distro
   package manager without rebuilding the entire application binary.

Distroless images and scratch provide neither a package manager nor an independently
updateable `tzdata` package. Adding `ca-certificates` and `tzdata` to scratch requires
embedding them in the binary or copying them manually, eliminating the simplicity argument
and making future updates harder.

## Decision

We deliberately do NOT use distroless or scratch as the runtime base image.

The `runtime` stage is based on `alpine:3.23`. The image setup:

1. Runs `apk upgrade --no-cache` to apply upstream security patches at build time.
2. Installs `ca-certificates` (TLS trust) and `tzdata` (timezone data) — the only two
   OS-level runtime dependencies.
3. Creates a dedicated non-root group and user (`app:app`) and switches to that user before
   the `COPY` and `CMD` instructions, so the binary never runs as root.

Setup teams reviewing this ADR should confirm whether the `alpine` pin (`3.23`) should be
updated and whether base image digests should be added for reproducibility (the Dockerfile
notes recommend digest pinning for production).

## Consequences

### Positive Consequences

- `apk upgrade` at build time keeps the OS layer current without waiting for a new Alpine
  minor release.
- `ca-certificates` and `tzdata` are managed by the distro, not embedded in the binary,
  so they can be updated independently.
- Non-root execution reduces the blast radius of a container escape.

### Negative Consequences

- Alpine is a larger base than distroless or scratch. The difference is a few megabytes for
  a statically linked binary.
- Alpine uses musl libc; CGO is disabled (`CGO_ENABLED=0`) in the builder, so no musl
  compatibility issues arise in practice.

## Alternatives Considered

### distroless/static or distroless/base

No shell, smaller surface — but no `apk` for patching, no convenient `tzdata` package, and
`ca-certificates` must be copied manually. Complicates future maintenance without a
meaningful security gain given that CGO is already disabled.

### scratch

Absolute minimum size. No CA bundle, no timezone data, no package manager. Requires manual
copy steps that are hard to keep up to date and breaks any code path that calls TLS or
resolves timezones at runtime.

## Notes

- `docker/server/Dockerfile` lines 42-53 define the `runtime` stage.
- `docker/server/README.md` §"runtime" describes the non-root user setup and the two
  OS-level packages.
- Production deploys should pin the base image by digest (noted in the Dockerfile header).
- Source: `docker/server/Dockerfile`, `docker/server/README.md`.
