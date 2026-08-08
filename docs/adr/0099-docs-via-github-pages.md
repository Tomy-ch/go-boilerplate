---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [docs, deploy]
---

# ADR-0099: Publish static docs/ via GitHub Pages (released on production push)

## Status

accepted

## Context

The repository maintains a `docs/` directory as the canonical source for architecture
documentation, design decisions, API references, and guides (see ADR-0008). This content
is useful to contributors and adopters when browsable in a rendered form, not just as raw
Markdown in the repository.

A hosting solution is needed that requires no additional infrastructure, aligns with the
existing GitHub-hosted workflow, and publishes automatically when documentation changes
land in production.

## Decision

Publish the contents of `docs/` as a static site via GitHub Pages on every push to the
`production` branch that touches `docs/**` or the workflow file itself
(`.github/workflows/deploy-docs.yaml`).

The workflow:

1. **build** job: checks out the repository and uploads the `docs/` directory as a GitHub
   Pages artifact using `actions/upload-pages-artifact`.
2. **deploy** job: deploys the uploaded artifact to GitHub Pages using
   `actions/deploy-pages`, writing the resulting URL to the `github-pages` environment.

The workflow is triggered only on `production` pushes to paths under `docs/` or the
workflow definition itself. Non-documentation pushes to `production` do not trigger a Pages
deploy. The `concurrency` group `"pages"` prevents concurrent deploys; in-progress deploys
are not cancelled (`cancel-in-progress: false`) to avoid partial publishes.

Required permissions: `contents: read`, `pages: write`, `id-token: write`.

## Consequences

### Positive Consequences

- Documentation is browsable at a stable URL without any external hosting infrastructure.
- The deployment pipeline is minimal (two workflow steps) and entirely within GitHub's
  hosted runner ecosystem.
- Path filtering (`docs/**`) ensures documentation deploys are not triggered by unrelated
  code changes, keeping the Pages environment stable.
- Non-cancellable concurrent deploys (`cancel-in-progress: false`) prevent a partial
  publication from leaving the Pages site in a broken state.

### Negative Consequences

- GitHub Pages is a GitHub-specific feature; migrating to another hosting platform
  (e.g., Netlify, Cloudflare Pages) requires replacing the workflow.
- The published site reflects only the `production` branch; documentation on feature
  branches is not automatically previewed.
- Only the raw `docs/` directory is served; if a static-site generator is introduced in
  the future, the workflow must be extended to include a build step.

## Alternatives Considered

### External static hosting (Netlify, Cloudflare Pages, Vercel)

Provides branch previews and richer build pipelines, but introduces an external service
dependency. Inconsistent with the vendor-neutral posture described in
[ADR-0001](0001-avoid-lock-in.md) when a built-in GitHub feature meets the requirement.

### Manual publishing

Eliminates automation overhead but risks stale published docs and requires human
coordination for every documentation change. Rejected.

## Notes

- `.github/workflows/deploy-docs.yaml` lines 1-44 are the complete workflow definition.
- The canonical-source documentation principle is ADR-0008.
- Source: `.github/workflows/deploy-docs.yaml`.
