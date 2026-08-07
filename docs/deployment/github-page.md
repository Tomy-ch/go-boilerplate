
# GitHub Pages

This project publishes documentation with GitHub Pages.
Deployment is executed through GitHub Actions.

## Overview

GitHub Pages for this repository is operated with the following policy:

- Publishing method: GitHub Actions
- Repository setting: `Settings > Pages > Source = GitHub Actions`
- Deployment definition: `.github/workflows/deploy-docs.yaml`
- Source of truth for deployment behavior: the GitHub Actions workflow

> [!NOTE]
> This document explains the operational policy and points to check for GitHub Pages.
> The exact build and deployment steps must follow `.github/workflows/deploy-docs.yaml`.

## Why GitHub Actions is used

This repository does not use the legacy "Deploy from a branch" Pages configuration.
Instead, documentation is built and deployed through GitHub Actions so that the publishing process stays reproducible and reviewable.

Advantages:

- The deployment procedure is version-controlled
- Build inputs are visible in the workflow file
- Local file placement and repository settings are less likely to drift
- It is easier to extend the pipeline later

## Repository Settings

Check the following repository setting before troubleshooting:

### Settings > Pages

- Source: `GitHub Actions`

If this is not set correctly, the workflow may succeed while the site is not published as expected.

## Workflow

The GitHub Pages deployment workflow is defined here:

```text
.github/workflows/deploy-docs.yaml
```

When you need the exact trigger, build steps, artifact path, or deploy steps, always check the workflow file first.

## Base Path

GitHub Pages for a project repository is published under the repository path.

```text
https://<username>.github.io/<repository-name>/
```

Example:

```text
https://example-org.github.io/example-api/
```

Because of this, documentation assets should be referenced with care.

### Recommended

- Use relative paths where possible
- Keep links and static asset references compatible with the repository base path

### Avoid

- Root-relative paths that assume `/` is the site root

Example:

```html
<!-- NG -->
<link rel="stylesheet" href="/styles.css">

<!-- OK -->
<link rel="stylesheet" href="./styles.css">
```

## Notes

### Single source of truth

If this document and the workflow differ, treat `.github/workflows/deploy-docs.yaml` as the source of truth.

### Static site assumptions

GitHub Pages serves static files.
If a page depends on runtime server behavior, it will not work as-is on Pages.

### SPA routing

If a single-page application is published on GitHub Pages, reloading a deep link may return `404`.
If SPA routing is introduced, add an appropriate fallback strategy such as a `404.html` redirect.

### Cache delay

GitHub Pages may continue showing cached content for a short time after deployment.
If changes do not appear immediately, check the workflow result first and then retry with a hard refresh.

## Troubleshooting

### The site is not updated

Check the following in order:

1. `.github/workflows/deploy-docs.yaml` was triggered
2. The workflow completed successfully
3. `Settings > Pages > Source` is set to `GitHub Actions`
4. The published path and asset paths are compatible with the repository base path

### Assets or links are broken

Typical causes:

- Using root-relative paths such as `/assets/...`
- Assuming the site is published at the domain root
- A mismatch between generated paths and the GitHub Pages repository path

### The deployment behavior is unclear

Open the workflow file and confirm:

- trigger conditions
- build commands
- output directory or uploaded artifact
- Pages deploy step

## Related Files

- `.github/workflows/deploy-docs.yaml`
- `docs/`
- `docs/portal/`

## Operational Guideline

When changing documentation publishing behavior, update both of the following together:

1. `.github/workflows/deploy-docs.yaml`
2. this document

This keeps the documented operation aligned with the actual deployment behavior.
