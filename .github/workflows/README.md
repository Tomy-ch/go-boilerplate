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
| Assistant | `@claude` mentioned on a pull request | Answer or investigate on demand, restricted to accounts with write access |

The `gen-*-artifacts-check` workflows protect the invariant "the committed generated output is reproducible from the generator". That invariant breaks in two directions — the generator inputs change without the output being regenerated, and the output is edited directly — so their `on.pull_request.paths` must list **both the inputs and the generated output**. Watching only the inputs makes the check structurally blind to a PR that touches the output alone, which lets the broken artifact reach the base branch and turns the next unrelated PR red instead.

Because these workflows pin their generators through `mise.toml`, that file is an input to most of them. A `paths` filter matches whole files, so a bump to any unrelated tool in that shared lockfile also triggers them — including the Postgres-backed `gen-db` job. That over-triggering is accepted deliberately: a generator version bump is exactly the change that should be re-verified, and splitting `mise.toml` to narrow the trigger would cost more than the occasional extra run.

## Workflow List

### CI Checks (Pull Request)

|Workflow|File|Description|
|---|---|---|
|Go Lint|`go-lint.yaml`|Run golangci-lint on Go code|
|Go Test|`go-test.yaml`|Run Go tests with coverage reporting, plus the `scripts/` tool tests outside the coverage gate|
|Module Tidy Check|`tidy-check.yaml`|Verify go.mod / go.sum are tidied|
|SQL Lint|`sql-lint.yaml`|Run sqlfluff on migration / DML / seed SQL files|
|Actions Lint|`actions-lint.yaml`|Run actionlint on workflow definitions, shellcheck the `run:` scripts of composite actions, plus the PR-comment secret and fence checks|
|Migration Check|`migration-check.yaml`|Validate migration files (duplicates, gaps, up/down pairing)|
|Sync Versions Check|`sync-versions-check.yaml`|Verify mise.toml versions are propagated to go.mod / Dockerfiles / READMEs|
|Generated Go Artifacts Check|`gen-go-artifacts-check.yaml`|Verify generated Go code matches committed artifacts|
|Generated Database Artifacts Check|`gen-db-artifacts-check.yaml`|Verify generated sqlc code matches committed artifacts|
|Generated OpenAPI Artifacts Check|`gen-oapi-artifacts-check.yaml`|Verify OpenAPI bundle and docs match committed artifacts|
|Generated Mock-Auth OpenAPI Artifacts Check|`gen-mock-auth-oapi-artifacts-check.yaml`|Verify the mock-auth-server OpenAPI bundle, zod schemas, and docs match committed artifacts|
|Mock-Auth Server Check|`mock-auth-server-check.yaml`|Type-check the mock-auth-server, run its unit / integration tests, and fail on golden JWKS fixture drift|
|OpenAPI Lint|`oapi-lint.yaml`|`redocly lint` the OpenAPI definition (naming / casing / descriptions / unused components)|
|App Boot Check|`app-di-startup-check.yaml`|Verify the application server starts successfully with DB|
|Job Boot Check|`job-boot-check.yaml`|Verify the job entrypoint boots and rejects an unknown job|
|Worker Boot Check|`worker-boot-check.yaml`|Verify the worker entrypoint boots (DI / DB) and rejects an unknown worker|
|Dockerfile Lint|`docker-lint.yaml`|Run hadolint on Dockerfiles (via go_tool_runner)|
|Skill Definition Lint|`md-skill-lint.yaml`|Check the `.claude/**` skill / agent definitions against reality and their `.codex/**` counterparts for existence parity|
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
|Config Scan|`trivy-config.yaml`|Trivy misconfiguration scan of the Dockerfiles, gating at HIGH|
|SAST|`sast.yaml`|Opengrep (Semgrep-compatible) scan of first-party source with taint tracking|
|Lockfile Integrity|`lockfile-integrity.yaml`|Verify every npm `resolved` URL points at the official registry over HTTPS|
|OpenAPI Security|`openapi-security.yaml`|Spectral with the OWASP API Security ruleset over the OpenAPI definition|
|Fuzz|`fuzz.yaml`|Go native fuzzing over the parsers that accept external text|
|Capability Diff|`capability-diff.yaml`|capslock report of capability changes in the Go dependency graph (report-only)|
|Notify|`notify.yaml`|Reusable `workflow_call` target that pushes a scheduled failure, or a finding from a non-blocking scanner, to a human|

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
| npm cooldown audit | lockfile / `.npmrc` changes | same as above | weekly |
| Trivy config (misconfig) | Dockerfile-change PRs | same as above | — |
| Trivy licence | same trigger as Trivy FS | same as above | weekly |
| OSV diff | dependency-change PRs | — | — |
| Opengrep (SAST) | Go / dependency / spec-change PRs | same as above | weekly |
| lockfile-lint | lockfile-change PRs | — | — |
| Spectral (OpenAPI) | spec-change PRs | `release/*` / deploy branches | — |
| capslock | `go.mod`-change PRs | — | — |
| Go fuzzing | — | — | weekly |

Weekly runs are staggered across Monday, one scanner per hour, so a single hour does not queue every scanner at once: `0 0` Trivy FS, `0 1` govulncheck, `0 2` TruffleHog, `0 3` OSV-Scanner, `0 4` Scorecard, `0 5` CodeQL, `0 6` Image Scan, `0 7` gitleaks (full-history), `0 8` zizmor (online audits), `0 9` npm cooldown audit, `0 10` Opengrep, `0 11` fuzz.

Every scanner with a weekly schedule calls `notify.yaml` when its job ends in `failure` or `cancelled`. A PR failure is already visible to its author; a scheduled failure is visible to nobody, which is the case the notification exists for. `cancelled` is included because a job killed by a timeout or a runner fault reports that rather than `failure`.

Failure is not the only thing worth pushing. A report-only scanner leaves its job green on a finding, so failure mode can never fire for one; those call `notify.yaml` in detection mode instead, which names the actor, ref, commit and the findings themselves. Both modes skip delivery and leave the run green when no webhook secret is configured, so a fork is never failed by a notification it cannot send.

Which trigger a detection notification fires on follows from who the right recipient is. For the vulnerability scanners it is the scheduled run only — on a PR the finding is already in a comment addressed to the author, who introduced the dependency, whereas a weekly finding is a newly published advisory against code that stood still and reaches nobody. The npm cooldown audit is the exception and fires on every trigger, because the decision to bypass the cooldown belongs to a tech lead / architect who is not necessarily on the PR.

| Workflow | Fires when | Trigger |
| --- | --- | --- |
| `npm-cooldown-audit.yaml` | any cooldown finding | all |
| `trivy-fs.yaml` | fixable CRITICAL / HIGH / MEDIUM found | schedule |
| `vulnerability-check.yaml` | reachable vulnerability found | schedule |
| `osv-scanner.yaml` | promotion-blocking finding | schedule |

The other scheduled scanners need no detection notification: gitleaks, TruffleHog, Opengrep, zizmor (at high), the image-scan gate and fuzzing all fail their job on a finding, so failure mode already delivers it. Three are deliberately left unconnected: the Trivy licence inventory reports licences nobody has yet agreed are problems (the same reason it writes no SARIF), while CodeQL and Scorecard publish to the code-scanning dashboard and expose no finding count to the workflow — a Scorecard "score dropped" notification would additionally need the previous score kept somewhere, which nothing here does.

#### Overlapping surfaces

Several tools can detect the same class of finding. Each surface has one owner so that a single problem does not gate twice and get suppressed twice:

| Surface | Owner | Also capable, deliberately not used here |
| --- | --- | --- |
| Dockerfile security policy | `trivy-config.yaml` | Opengrep (its Dockerfile rules are excluded in `sast.yaml`) |
| Dockerfile style / correctness | `docker-lint.yaml` (hadolint) | — (a different layer, not a duplicate) |
| First-party Go source | `sast.yaml` (Opengrep) + `gosec` in golangci-lint | — |
| OpenAPI conventions / naming | `oapi-lint.yaml` (redocly) | Spectral |
| OpenAPI security posture | `openapi-security.yaml` (Spectral) | redocly |

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

### Assistant (Comment)

|Workflow|File|Trigger|Description|
|---|---|---|---|
|Claude|`claude.yaml`|`@claude` in a pull-request comment or review|Run Claude Code against the pull request on demand|

## Shared Composite Actions

Reusable composite actions live under [`.github/actions/`](../actions/):

|Action|Purpose|
|---|---|
|`setup-postgres`|Wait for and initialize the Postgres service container (used by DB-dependent jobs)|
|`upsert-pr-comment`|Marker-based PR comment upsert (detect existing → update / create) with a shared Commit / UpdatedAt footer, used by the result-commenting workflows|
|`osv-scan`|Run osv-scanner and classify each finding against the release-gate severity policy, shared by the OSV reporting workflow and the OSV release gate|

## Notes

- Comments and log messages in `.github/workflows/**` and `.github/actions/**` are written in
  **English**, including `echo` output and `::error::` annotations. The repository's Japanese-comment
  convention covers Go code, test names, PRs, and replies — it does not extend to CI definitions,
  whose readers are the workflow logs and the wider Actions ecosystem. The content standard in
  [`docs/rules.md`](../../docs/rules.md) § Comment Rules still applies: no how-narration, no
  development history, no restatement, and keep a non-obvious Why.
- `auto-generate-docs.yaml` opens an auto-PR whose branch is named `auto/docs-update/<base>` (one branch per release base, reused across runs with `delete-branch: true`); the workflow skips itself on that branch to avoid recursion.
- All deployment workflows require their target branch (`production` / `staging` / `develop`) to be branch-protected; merges must flow through PR review.
- Security scan triggers are defined per tool in the Security Trigger Matrix above; if a high-severity CodeQL or Trivy finding appears, the corresponding branch-protection rule blocks merge.
- `trivy-fs.yaml` and `osv-scanner.yaml` never fail a check: every finding, fixed or not, is written to code scanning and to the PR comment, and the blocking verdict is left to the release gates described above, so a promotion cannot silently ship a known vulnerability while an ordinary PR is not held hostage to one it did not introduce.
- `trufflehog.yaml` reports only *verified* secrets and never writes a raw secret value into the job log, the PR comment, or an artifact; gitleaks covers the regex-based side with `--redact`.
- **A job that comments on a PR is passed no secret.** Secret masking only covers the path where the runner captures job output for the log; bytes a step wrote to a file with `tee` never pass through it, and `upsert-pr-comment` reads its body from exactly such a file. A value that looks masked in the log therefore lands raw in a public comment. No inspection step currently receives a secret, and `make actions-comment-secret-lint` keeps it that way by failing when a job using the action is passed anything other than `GITHUB_TOKEN`. Needing a secret in an inspection step means splitting it into a job that does not comment. The check reads direct `secrets` references only, so routing one through `needs.<job>.outputs` would evade it — the rule, not the linter, is what holds.
- **`upsert-pr-comment` matches its own comment on a bot author and a leading marker.** Anyone can post a comment carrying the marker on a public repository, and every workflow here comments under the same bot, so neither the marker nor the author identifies a comment on its own: a pull-request author who gets one workflow to echo another's marker into its log would otherwise steer that other workflow onto the wrong comment. A body the action wrote always opens with the marker, which a planted one never does. `github-token` must consequently be a token that posts as a bot — `GITHUB_TOKEN` or a GitHub App token; a PAT posts as a user, whose comment the action would never find, leaving a new comment on every run.
- **A fence around attacker-controlled text is sized from that text, never fixed.** `upsert-pr-comment` computes its fence as one backtick longer than the longest run in the body, but only on the `details-summary` path; without that input the body is passed through untouched, because several callers write Markdown — headings, tables, their own `<details>` — that is meant to render. A caller on that path which fences part of its body itself therefore owns the fence, and a fixed three-backtick one is closable by any body that reproduces source lines: a linter quoting a file the pull-request author wrote lands three backticks inside the block, and everything after renders as live Markdown under the bot's name. `sql-lint.yaml` builds its fences this way and sizes each from its own log; `capability-diff.yaml` leaves fencing to the action and wraps only its step summary. Callers must also keep each body under the action's `max-length`, which is applied *before* fencing — a body trimmed there loses its closing fence. `make actions-comment-fence-lint` covers the two parts that are mechanically decidable — no literal fence is emitted from a `run:` block, and the duplicated `fence_for` helpers stay identical — but whether a body is attacker-controlled is not, so the rule is what holds.
- Exceptions to zizmor's audits live in `.github/zizmor.yml`. `ignore` there is file-scoped on purpose, so a new workflow hitting the same audit still fails; entries are removed as the underlying finding is fixed rather than kept as a permanent allowlist.
- **Cache safety.** Caches are branch-scoped — a run restores only its own ref's cache plus the default branch's — so a pull-request run cannot write a cache that a later `release/*` push restores. That is why caching stays enabled on ordinary CI workflows. Poisoning becomes possible in two places. First, when untrusted PR code executes in a trusted scope while a cache is saved: `pull_request_target` and `workflow_run` run in the base ref's scope, so a workflow that checks out the PR head there would leave its cache exactly where privileged runs read it. Never combine those two; a workflow that must handle untrusted code keeps caching off. Second, **between workflows that share a branch scope but not a privilege level** — several ordinary workflows also run on pushes to protected branches, so a compromised one could leave a poisoned tool cache for a job holding `security-events: write` to restore and execute. Every job with that permission therefore sets `cache: false`, trading a slower install for not inheriting an artifact a lower-privileged run could have written.
- The `Detect changes` step in `auto-generate-docs.yaml` excludes coverage HTML and SchemaSpy timestamp churn so cosmetic regenerations do not open noise PRs.
- GitHub disables scheduled workflows automatically after 60 days without a commit, and it does so silently. Keeping them alive is out of scope for this template — no keepalive job is provided — so a fork that goes quiet should expect to re-enable them from the Actions tab.
- A repository created from a fork or template starts with every workflow in `disabled_fork` state, where nothing runs at all. `make enable-workflows` enumerates and enables them; it is idempotent and safe to re-run.
- **`claude.yaml` authorization.** Who may invoke Claude is decided by the action's own write-permission check, not by an allowlist in the workflow. Both alternatives break under forking: a hardcoded list of accounts locks a fork owner out of their own repository, and a repository variable holding that list is never inherited by a fork, so it resolves empty and nobody can invoke anything. A permission check resolves against whichever repository the workflow runs in and therefore needs no configuration to be correct anywhere. Two inputs would undo this and are deliberately left unset: `allowed_non_write_users` bypasses the check outright, and `allowed_bots` admits Apps that need neither installation nor write access. The workflow's own `if:` reads only the `github` context, so no comment starts a runner; it grants no authority. Restricting *who* cannot address prompt injection carried in a fork pull request — the invoker is trusted, the diff Claude reads is not — which is why `contents` stays read-only.
- `.spectral.yaml` and `.trivyignore.yaml` follow the same policy as `.github/zizmor.yml`: nothing is disabled in bulk, every entry carries the ADR or implementation that justifies it, and suppressions are scoped to a path (or a JSON pointer) so a new file hitting the same rule still fails.
- `fuzz.yaml` is scheduled rather than run per PR: a fuzz run explores a random corpus, so gating a merge on it would make the verdict depend on a coin flip. Crash reproducers are committed under `testdata/fuzz/` and replay as ordinary regression tests.
