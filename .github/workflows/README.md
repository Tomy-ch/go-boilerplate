# GitHub Actions Workflows

English | [日本語](README.ja.md)

This directory contains GitHub Actions workflow definitions for CI/CD. Workflows are grouped by purpose: pull-request gates (lint / test / security scans), push-triggered deployments, and documentation regeneration on release branches.

## Trigger Strategy

| Group | When it runs | What it does |
| --- | --- | --- |
| CI Checks | every pull request | Block merge if lint / test / generated-artifact consistency fails |
| Security | per-tool matrix (see below) | Surface vulnerabilities in code, dependencies, images, workflow definitions, and committed secrets |
| Deployment | push to `production` / `staging` / `develop` | Build artifacts, run migration, deploy app or docs portal |
| Documentation | push to `release/*` | Regenerate OpenAPI / ER / portal docs and open an auto-sync PR |

## Workflow List

### CI Checks (Pull Request)

|Workflow|File|Description|
|---|---|---|
|Go Lint|`go-lint.yaml`|Run golangci-lint on Go code|
|Go Test|`go-test.yaml`|Run Go tests with coverage reporting|
|Module Tidy Check|`tidy-check.yaml`|Verify go.mod / go.sum are tidied|
|SQL Lint|`sql-lint.yaml`|Run sqlfluff on migration / DML / seed SQL files|
|Actions Lint|`actions-lint.yaml`|Run actionlint on workflow / composite-action definitions (via go_tool_runner)|
|Migration Check|`migration-check.yaml`|Validate migration files (duplicates, gaps, up/down pairing)|
|Sync Versions Check|`sync-versions-check.yaml`|Verify mise.toml versions are propagated to go.mod / Dockerfiles / READMEs|
|Generated Go Artifacts Check|`gen-go-artifacts-check.yaml`|Verify generated Go code matches committed artifacts|
|Generated Database Artifacts Check|`gen-db-artifacts-check.yaml`|Verify generated sqlc code matches committed artifacts|
|Generated OpenAPI Artifacts Check|`gen-oapi-artifacts-check.yaml`|Verify OpenAPI bundle and docs match committed artifacts|
|OpenAPI Lint|`oapi-lint.yaml`|`redocly lint` the OpenAPI definition (naming / casing / descriptions / unused components)|
|App Boot Check|`app-di-startup-check.yaml`|Verify the application server starts successfully with DB|
|Job Boot Check|`job-boot-check.yaml`|Verify the job entrypoint boots and rejects an unknown job|
|Worker Boot Check|`worker-boot-check.yaml`|Verify the worker entrypoint boots (DI / DB) and rejects an unknown worker|
|Dockerfile Lint|`docker-lint.yaml`|Run hadolint on Dockerfiles (via go_tool_runner)|
|Pin Actions Check|`pin-actions-check.yaml`|Verify GitHub Actions are pinned to a SHA (supply-chain hardening)|
|Pin Images Check|`pin-images-check.yaml`|Verify Docker base images are pinned to a digest per the lockfile (supply-chain hardening)|

### Security

|Workflow|File|Description|
|---|---|---|
|CodeQL Scan|`code-ql.yaml`|CodeQL analysis for security vulnerabilities|
|Dependency Scan|`trivy-fs.yaml`|Trivy filesystem scan for library vulnerabilities (developer-facing)|
|Release Dependency Scan|`trivy-release-gate.yaml`|Trivy filesystem scan on PRs into develop/staging/production|
|Image Scan|`image-scan.yaml`|Build image, generate SBOM, run Trivy scan|
|Vulnerability Scan|`vulnerability-check.yaml`|govulncheck for actionable Go vulnerabilities|
|OSV Scan|`osv-scanner.yaml`|OSV database scan across the Go module graph and the npm lockfiles|
|Release OSV Scan|`osv-release-gate.yaml`|OSV scan on PRs into develop/staging/production, failing on HIGH or above|
|Secret Scan|`secret-scan.yaml`|gitleaks scan for committed secrets (via go_tool_runner)|
|Secret Scan (TruffleHog)|`trufflehog.yaml`|TruffleHog scan for *verified* secrets — credentials that are actually live|
|Actions Static Analysis|`zizmor.yaml`|zizmor audit of the workflow / composite-action definitions themselves|
|Dependency Review|`dependency-review.yaml`|Block a PR that introduces a newly vulnerable dependency|
|OpenSSF Scorecard|`scorecard.yaml`|Score the repository's security posture and publish the result|
|npm Cooldown Audit|`npm-cooldown-audit.yaml`|Report lockfile entries that do not satisfy the `.npmrc` supply-chain cooldown (never blocks)|

Every scanner writes SARIF to GitHub code scanning where it can, and comments its result on the PR through the shared `upsert-pr-comment` action.

#### Security Trigger Matrix

Each tool runs where its findings can actually change: a PR surfaces the risk the change itself introduces, a push to a protected branch keeps a code-scanning baseline for branch protection to judge, and a weekly schedule only exists for tools whose result can change while the code stands still (newly disclosed CVEs, new queries).

| Tool | Pull request | Push to protected branch | Schedule |
| --- | --- | --- | --- |
| gitleaks | all PRs | — | weekly, full history |
| TruffleHog | all PRs, diff only | — | weekly, full history |
| zizmor | when Actions files change | `develop` / `staging` / `production` / `release/*` | weekly (online audits) |
| Dependency Review | dependency-change PRs | — | — |
| govulncheck | Go / dependency-change PRs | same as above | weekly |
| Trivy FS | Go / dependency-change PRs | same as above | weekly |
| OSV-Scanner | dependency-change PRs | same as above | weekly |
| CodeQL | Go / dependency-change PRs | same as above | weekly |
| OpenSSF Scorecard | — | default branch only | weekly |
| Image Scan | PRs into a deploy branch | — | weekly |
| Release gates (Trivy FS / OSV) | PRs into a deploy branch | — | — |
| npm cooldown audit | lockfile / `.npmrc` change | same as above | weekly |

Weekly runs are staggered across Monday, one scanner per hour, so a single hour does not queue every scanner at once: `0 0` Trivy FS, `0 1` govulncheck, `0 2` TruffleHog, `0 3` OSV-Scanner, `0 4` Scorecard, `0 5` CodeQL, `0 6` Image Scan, `0 7` gitleaks (full-history), `0 8` zizmor (online audits), `0 9` npm cooldown audit.

#### Release Gates

The dependency scanners are a double gate. On an ordinary PR they report only: a vulnerability inherited from the existing dependency tree is not something that PR introduced, and blocking there stops unrelated work while the update is prepared elsewhere. The blocking verdict happens on a PR into `develop` / `staging` / `production`, where the dependency state under review is the one about to be promoted.

| Gate | Fails on |
| --- | --- |
| `trivy-release-gate.yaml` | any Trivy finding, including one with no released fix |
| `osv-release-gate.yaml` | any OSV finding rated HIGH or CRITICAL, fixed or not, plus an unrated finding that has a fixed version |

Severity for the OSV gate comes from the advisory's own rating and falls back to the CVSS score osv-scanner aggregates per group. Advisories from the Go vulnerability database publish neither, so they cannot be measured against the HIGH threshold at all; those gate only when a fixed version exists, which keeps an advisory that can be neither rated nor updated away from turning every promotion permanently red. Both gates deliberately carry no `paths` filter — a promotion PR often changes no manifest, and a required check has to run to be able to block.

#### npm Cooldown Audit

Each `.npmrc` sets `min-release-age`, npm's own supply-chain quarantine: a version published inside that window cannot be resolved at all. The catch is that it applies only while npm *resolves* dependencies. Every CI job and image build here runs `npm ci`, which replays the lockfile without resolving, so a lockfile produced with the cooldown switched off installs cleanly and shows no symptom anywhere in CI.

`npm-cooldown-audit.yaml` closes that blind spot. It reads the window from each lockfile's own `.npmrc` rather than hardcoding it, and reports any entry younger than the window. The signal is close to noise-free: under an active cooldown npm refuses to resolve an in-window version, so the only way one reaches a lockfile is a deliberate override.

**It never fails the build**, and that is a design decision rather than a default. Overriding the cooldown is a tech-lead / architect call — reacting to a CRITICAL advisory is the case it exists for — so a hard gate would block precisely the legitimate use. The non-blocking property lives in the tool itself, not in workflow configuration, so it cannot be turned into a gate by editing YAML.

Its scope is honest but narrow: **policy drift** — accidents, convention rot, a change in npm's own behaviour. It is not a defence against someone with commit access, who can delete the workflow in the same change. What it provides there is detection and attribution, with deterrence left to the organisation. The enforcement half is [`CODEOWNERS`](../CODEOWNERS), which reserves review of `**/.npmrc`, `**/package-lock.json`, and the pin lockfiles to the owning role.

A pull request is audited against its base, so a finding names exactly the entries that change introduces and the PR comment persists as the record even after those versions age out of the window. The scheduled run audits every entry as a second net.

#### Runner Hardening

Every job in this directory starts with `step-security/harden-runner` in `egress-policy: audit` mode. It records the runner's outbound network calls and file-integrity events so a compromised action or transitive tool download becomes visible. `audit` only reports — moving to `block` requires a settled allowlist of endpoints, which is deliberately deferred until the audit data exists.

### Deployment (Push)

|Workflow|File|Trigger|Description|
|---|---|---|---|
|Deploy App|`deploy-app.yaml`|push to production/staging/develop|Build and push Docker images (image signing via cosign + provenance / SBOM attestation), run migration and deploy|
|Deploy Docs|`deploy-docs.yaml`|push to production (docs changes)|Deploy documentation portal to GitHub Pages|

### Documentation (Push)

|Workflow|File|Trigger|Description|
|---|---|---|---|
|Auto-generate Docs|`auto-generate-docs.yaml`|push to release/* branches|Sync OpenAPI `info.version` from the `release/vX.Y.Z` branch name, then auto-generate the OpenAPI bundle / embedded spec / docs, ER diagrams, portal docs|

## Shared Composite Actions

Reusable composite actions live under [`.github/actions/`](../actions/):

|Action|Purpose|
|---|---|
|`setup-postgres`|Wait for and initialize the Postgres service container (used by DB-dependent jobs)|
|`upsert-pr-comment`|Marker-based PR comment upsert (detect existing → update / create) with a shared Commit / UpdatedAt footer, used by the result-commenting workflows|
|`osv-scan`|Run osv-scanner and classify each finding against the release-gate severity policy, shared by the OSV reporting workflow and the OSV release gate|

## Notes

- `auto-generate-docs.yaml` opens an auto-PR whose branch is named `auto/docs-update/<base>` (one branch per release base, reused across runs with `delete-branch: true`); the workflow skips itself on that branch to avoid recursion.
- All deployment workflows require their target branch (`production` / `staging` / `develop`) to be branch-protected; merges must flow through PR review.
- Security scan triggers are defined per tool in the Security Trigger Matrix above; if a high-severity CodeQL or Trivy finding appears, the corresponding branch-protection rule blocks merge.
- `trivy-fs.yaml` and `osv-scanner.yaml` never fail a check: every finding, fixed or not, is written to code scanning and to the PR comment, and the blocking verdict is left to the release gates described above, so a promotion cannot silently ship a known vulnerability while an ordinary PR is not held hostage to one it did not introduce.
- `trufflehog.yaml` reports only *verified* secrets and never writes a raw secret value into the job log, the PR comment, or an artifact; gitleaks covers the regex-based side with `--redact`.
- Exceptions to zizmor's audits live in `.github/zizmor.yml`. `ignore` there is file-scoped on purpose, so a new workflow hitting the same audit still fails; entries are removed as the underlying finding is fixed rather than kept as a permanent allowlist.
- zizmor's `cache-poisoning` audit fires on almost every workflow here — a caching `setup-*` action plus a trigger it associates with publishing — and is remapped to `medium` rather than answered file by file. Caches are branch-scoped: a run restores only its own ref's cache and the default branch's, so a pull-request run cannot write a cache that a later `release/*` push restores. Setting `cache: false` on one workflow therefore removes the alert without removing any risk, and leaves an exception nothing else follows. What would make the audit real is a `pull_request_target` / `workflow_run` workflow that checks out the PR head **and** saves a cache: those run in the base ref's scope, so untrusted code's cache lands exactly where privileged runs read it. There is no such workflow here — do not add one.
- The `Detect changes` step in `auto-generate-docs.yaml` excludes coverage HTML and SchemaSpy timestamp churn so cosmetic regenerations do not open noise PRs.
