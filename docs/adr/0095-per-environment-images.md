---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [deploy, image, config]
---

# ADR-0095: Per-environment images (.env matrix x APP_ENV build-arg, fixed at build time)

## Status

accepted

## Context

The application reads configuration from an `.env` file that is embedded into the binary at
build time via `go:embed` (see ADR-0042). Because the embedded file is baked in at compile
time, a single binary can only carry one environment's configuration. Separate per-environment
configurations (production, staging, development) therefore require separate images.

At the same time, injecting environment-specific secrets or config values at container
runtime (via env vars or mounted files) must still be possible; the embedded config provides
the safe defaults, while runtime env vars override them.

The CI/CD pipeline runs on three long-lived branches — `production`, `staging`, and `develop`
— that map 1:1 to environments.

## Decision

Each branch push produces an image that embeds the matching `.env.<env>` file, selected by
the `APP_ENV` build argument:

| Branch | APP_ENV | Embedded config |
| --- | --- | --- |
| `production` | `prd` | `env/.env.prd` |
| `staging` | `stg` | `env/.env.stg` |
| `develop` | `dev` | `env/.env.dev` |

The `builder` stage materializes the target config before compilation:

```text
cp "env/.env.${APP_ENV}" env/.env
go build ... -o /app/bin/server ./cmd/
```

The `Define build environment` step in the workflow maps `github.ref_name` to `app_env` and
passes it as a `build-arg` to the Docker build. Runtime environment variables can still
override any embedded value, so secrets are never required to be embedded.

## Consequences

### Positive Consequences

- The environment a container runs in is determined at build time, not runtime injection,
  eliminating a class of misconfiguration errors (wrong env vars, missing env vars at
  container start).
- Each image is independently verifiable and auditable: the embedded config is part of
  the signed artifact (see [ADR-0097](0097-release-image-supply-chain.md)).
- No runtime config-management sidecar or secret-injection step is required for the base
  config values.

### Negative Consequences

- Config changes for a given environment require a full rebuild and push of that
  environment's image.
- Three distinct image tags must be maintained (one per environment), increasing registry
  storage.
- Secrets must never be placed in `env/.env.<env>` files (they would be embedded in the
  image layer). Runtime env var injection is the appropriate path for secrets.

## Alternatives Considered

### Single image with runtime config injection

One image for all environments; env vars or mounted files supply config at runtime. Simpler
registry, but requires a reliable config-injection mechanism (secret manager, mounted
ConfigMap/Secret) at every deployment target and makes the "what config is this container
running?" question harder to answer from the image alone.

### Per-environment Dockerfile

A separate Dockerfile per environment. Duplicates build and maintenance effort; the
`APP_ENV` build arg achieves the same outcome from a single Dockerfile without duplication.

## Notes

- `docker/server/Dockerfile` lines 25-30: `ARG APP_ENV=prd` and the `cp` command.
- `.github/workflows/deploy-app.yaml` `Define build environment` step (lines 54-63): branch
  → `app_env` mapping.
- `env/README.md`: "Env files are embedded into the binary at build time (`embed.go`). The
  Docker `builder` stage materializes the target via the `APP_ENV` build arg."
- The embedded-config mechanism itself is documented in ADR-0042.
- Source: `docker/server/Dockerfile`, `.github/workflows/deploy-app.yaml`, `env/README.md`.
