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

## Result Comments

A pull request here runs some thirty checks and nearly every one of them can comment. A comment per passing check states what no one asked and buries the one comment that has something to say, so a result comment is **created only when the check has something to report**. The `status` input on `upsert-pr-comment` carries that verdict: the literal `success` is the only value that suppresses a comment, and every other value — including the empty string a cut-off job leaves behind — posts one.

Silence is not how a fix gets reported. `success` suppresses *creating* a comment, never updating one, so a check that failed on an earlier push overwrites its own red comment with the green result, in place. The pull request never keeps a failure that no longer holds, and the reader learns it was resolved where the failure was reported rather than by noticing an absence. Deleting the comment instead would leave the fix unrecorded and, on a re-failure, drop the comment at the bottom of a thread that has moved past it.

**One comment is removed rather than overwritten: a cut-off notice.** It records no verdict — only that a run ended without reaching one — so a later `success` answers the question it left open instead of superseding a finding, and deleting it loses nothing. Overwriting would put a permanent green comment on the pull request in exchange for a cancellation, and cancellations are routine: `cancel-in-progress` kills the previous run on every quick second push, and a job cut off while piping its output leaves a half-written body, so the notice gets created and then frozen green. The cancellation stays in the run history either way. Recognition is by the `CUT OFF (no result produced)` heading — a job cut off mid-write has a body and so never reaches the action's own notice text, which leaves the heading every caller passes as the only common signal, and `make actions-cutoff-lint` is what keeps it mandatory. A refused delete falls back to the overwrite, so a green run never leaves a cut-off notice standing.

**The verdict is derived positively — only an explicit clean signal is quiet.** Where a step already publishes a `status` output, the caller passes it through; where the clean state is a count or a flag, the caller tests for that exact value (`steps.<id>.outputs.count == '0' && 'success' || 'findings'`) rather than testing for the finding. The difference shows up when the producing step never ran: the output is empty, which is not the clean value, so the check reports instead of going quiet. Reversing the test would make every unfinished run look like a pass. `✅` in a title and `success` here mean the same thing by construction — the scanners that mark a non-blocking finding `✅` (`osv-scan`) are quiet on it, and the ones that mark it `⚠️` (`sast`) are not.

Two callers pass a constant `report`, both in `image-scan.yaml`: an SBOM inventory and a Trivy table are not verdicts, so neither has a state that means "nothing to say", and that job only runs for a pull request into a deploy branch, where the contents of the image are what is under review.

A comment can outlive the run that wrote it. A `paths` filter means a later push may not run the workflow at all, leaving a red comment standing on a head it never examined; the Commit / UpdatedAt footer every comment carries is what distinguishes it from a current one, and the check run is what carries the authoritative status.

## Job Cut-off

A job can stop without reaching a verdict — a timeout, a cancellation, a runner fault. What a reader sees on the pull request in that case is not a property of the tool that was running; it follows from how the job and its comment step are declared, and every default here is the wrong one. `make actions-cutoff-lint` enforces the rules below, because keeping every comment step and every job in this directory correct by eye is not something a review can be asked to do.

**Every step that calls `upsert-pr-comment` must be reachable after a cancellation.** Actions prepends an implicit `success() &&` to any custom `if:` that contains no status-check function, so a cut-off job skips the comment step and the pull request keeps no trace at all — while the `Fail if …` step, which usually does carry `always()`, still turns the check red. Red with no readable reason is worse than either half alone. The condition therefore needs `always()` or `cancelled()`; `failure()` does **not** qualify, because it is false for a cancelled job. Every caller now uses the plain `always() && github.event_name == 'pull_request'`: whether a comment is warranted is decided by `status` inside the action, not by skipping the step, because a step that never runs cannot correct a comment an earlier push left behind.

**A missing body file is reported as a cut-off, not as a step failure.** A job cut off early never runs the step that writes the file, so absence is the normal shape of exactly the case the comment has to survive; `upsert-pr-comment` posts a cut-off notice and replaces the caller's heading, since no title the caller set can still describe a body that says the job never ran. The notice names no cause — a body can also be missing because an earlier step failed outright, and only the run log tells those apart. The cost is that a miswired `body-file` path surfaces as a cut-off notice on a green run rather than as a failure — loud enough to catch, which silence on a cut-off is not. Under `cancel-in-progress`, a superseded run may post that notice moments before the new run overwrites the same marker; that is the mechanism working, not a defect to fix.

**Absence is only half the test, so every caller passes a cut-off heading too.** Most inspection steps pipe their output straight into the body file with `tee` and set their `title` output only afterwards, from the exit code. Cut off mid-inspection, such a job leaves a *partially written* file behind — the action sees a body and cannot tell it from a finished one, while the title never got set. The caller therefore carries the other half of the signal as `${{ steps.<id>.outputs.title || '## ⚠️ <check>: CUT OFF (no result produced)' }}`: the fallback fires exactly when the producing step did not reach its own conclusion, and the partial log stays visible underneath it. Where the heading is a literal rather than a step output (`image-scan.yaml`, `sync-versions-check.yaml`), the same signal is expressed as a condition on that step's `outcome` / output. Note the GitHub-expression trap when writing one: `cond && '' || X` always yields `X`, because an empty string is falsy — the heading has to sit in the truthy branch.

**Every job sets `timeout-minutes`.** Without it a job runs to GitHub's 360-minute default, so one hang holds a runner for six hours. The value is the job's measured maximum × 3, rounded up to the next 5 minutes, with a floor of 10 to absorb setup variance on a contended runner; a job with no recent completed run gets 15. Only the values that depart from that formula are listed here — every other job is at the floor, and a value can be re-derived from the formula rather than looked up.

| Job | Minutes | Why not the formula |
| --- | --- | --- |
| `auto-generate-docs.yaml` `generate-docs` | 25 | measured ~7m |
| `go-test.yaml` `go-test` | 20 | measured ~5m |
| `image-scan.yaml` `build`, `deploy-app.yaml` `build` | 15 | image build with a cold layer cache varies well beyond its measured run |
| `deploy-app.yaml` `deploy` | 30 | a placeholder today; a real deployment wired in by a fork must not meet a 10-minute cap |
| `fuzz.yaml`, `scorecard.yaml`, `notify.yaml`, `osv-release-gate.yaml` | 15 | no recent completed run to measure |
| `dast.yaml` `dast` | 30 | no completed run to measure, and the job builds and boots the application before a scan whose length is set by the size of the OpenAPI definition <!-- dast:line --> |
| `code-ql.yaml` `codeql` | 30 | the limit covers whichever matrix leg is slowest, and no leg but `go` has a completed run to measure; `security-extended` is also a larger suite than the one the previous value was measured against |
| `secret-scan.yaml`, `trufflehog.yaml` | 15 | measured on pull requests only, where they scan a diff; the weekly run walks the full history and has never completed one to measure |
| `app-di-startup-check.yaml`, `gen-go-artifacts-check.yaml` | 15 | predate the formula; left as they are, since lowering a working limit only adds risk |
| `claude.yaml`, `go-lint.yaml`, `sample-removal-check.yaml` | 30 | as above; `go-lint` additionally runs golangci-lint with its own timeout disabled, so this is that job's only cutoff |

A job that starts tripping its limit has outgrown its measurement: re-measure and re-apply the formula rather than nudging the number. Jobs that call a reusable workflow cannot carry `timeout-minutes` at all — the key is invalid there — so the check skips them, and the limit lives in the called workflow's own job.

All three rules live in one check rather than three, because they are not three policies: a job with no limit is what produces a cut-off, and the two comment rules are what make one readable. Fixing any of them alone leaves the pull request no better off, so there is no case for running one without the others.

## Workflow List

### CI Checks (Pull Request)

|Workflow|File|Description|
|---|---|---|
|Go Lint|`go-lint.yaml`|Run golangci-lint on Go code|
|Go Test|`go-test.yaml`|Run Go tests with coverage reporting, plus the `scripts/` tool tests outside the coverage gate|
|Module Tidy Check|`tidy-check.yaml`|Verify go.mod / go.sum are tidied|
|SQL Lint|`sql-lint.yaml`|Run sqlfluff on migration / DML / seed SQL files|
|Actions Lint|`actions-lint.yaml`|Run actionlint on workflow definitions, shellcheck the `run:` scripts of composite actions, plus the PR-comment secret and fence checks and the job cut-off check|
|Migration Check|`migration-check.yaml`|Validate migration files (duplicates, gaps, up/down pairing)|
|Sync Versions Check|`sync-versions-check.yaml`|Verify mise.toml versions are propagated to go.mod / Dockerfiles / READMEs|
|Generated Go Artifacts Check|`gen-go-artifacts-check.yaml`|Verify generated Go code matches committed artifacts|
|Generated Database Artifacts Check|`gen-db-artifacts-check.yaml`|Verify generated sqlc code matches committed artifacts|
|Generated OpenAPI Artifacts Check|`gen-oapi-artifacts-check.yaml`|Verify OpenAPI bundle and docs match committed artifacts|
|Generated Mock-Auth OpenAPI Artifacts Check|`gen-mock-auth-oapi-artifacts-check.yaml`|Verify the mock-auth-server OpenAPI bundle, zod schemas, and docs match committed artifacts|
|Mock-Auth Server Check|`mock-auth-server-check.yaml`|Type-check the mock-auth-server, run its unit / integration tests, and fail on golden JWKS fixture drift|
|Portal Check|`portal-check.yaml`|Type-check the documentation portal viewer (`docs-viewer/`) and run its test suite|
|Scripts Check|`scripts-check.yaml`|Type-check the repository's TypeScript helper scripts (`scripts/**/*.ts`) and run the unit tests covering their decision logic|
|OpenAPI Lint|`oapi-lint.yaml`|`redocly lint` the OpenAPI definition (naming / casing / descriptions / unused components)|
|App Boot Check|`app-di-startup-check.yaml`|Verify the application server starts successfully with DB|
|Job Boot Check|`job-boot-check.yaml`|Verify the job entrypoint boots and rejects an unknown job|
|Worker Boot Check|`worker-boot-check.yaml`|Verify the worker entrypoint boots (DI / DB) and rejects an unknown worker|
|Dockerfile Lint|`docker-lint.yaml`|Run hadolint on Dockerfiles (via go_tool_runner)|
|Markdown Lint|`md-lint.yaml`|Lint Markdown shape with markdownlint, validate every ` ```mermaid ` fence with the real parser, and check the `.claude/**` skill / agent definitions against reality and their `.codex/**` counterparts|
|Commitlint|`commitlint.yaml`|Lint every commit message the PR adds to the base branch — the route the `commit-msg` hook cannot cover|
|Pin Actions Check|`pin-actions-check.yaml`|Verify GitHub Actions are pinned to a SHA (supply-chain hardening)|
|Pin Images Check|`pin-images-check.yaml`|Verify Docker base images are pinned to a digest per the lockfile (supply-chain hardening)|

### Security

|Workflow|File|Description|
|---|---|---|
|CodeQL Scan|`code-ql.yaml`|CodeQL analysis on the `security-extended` suite, one matrix leg per language: `go`, `javascript-typescript` (mock-auth-server / docs-viewer / scripts) and `actions` (the workflow definitions themselves)|
|Dependency Scan|`trivy-fs.yaml`|Trivy filesystem scan for library vulnerabilities (developer-facing)|
|Release Dependency Scan|`trivy-release-gate.yaml`|Trivy filesystem scan on PRs into develop/staging/production|
|Image Scan|`image-scan.yaml`|Build image, generate the SBOM in both SPDX-JSON and CycloneDX-JSON, run Trivy scan|
|Vulnerability Scan|`vulnerability-check.yaml`|govulncheck for actionable Go vulnerabilities|
|OSV Scan|`osv-scanner.yaml`|OSV database scan across the Go module graph and the npm lockfiles|
|Release OSV Scan|`osv-release-gate.yaml`|OSV scan on PRs into develop/staging/production, failing on HIGH or above|
|Secret Scan|`secret-scan.yaml`|Two independent scans of the working tree for committed secrets: gitleaks (wide regex / entropy net) and Trivy (curated rules, far fewer false positives), as separate jobs with separate verdicts|
|Secret Scan (TruffleHog)|`trufflehog.yaml`|TruffleHog scan for *verified* secrets — credentials that are actually live|
|Actions Static Analysis|`zizmor.yaml`|zizmor audit of the workflow / composite-action definitions themselves (same `make` gate as the pre-commit hook)|
|Dependency Review|`dependency-review.yaml`|Block a PR that introduces a newly vulnerable dependency|
|OpenSSF Scorecard|`scorecard.yaml`|Score the repository's security posture and publish the result|
|npm Cooldown Audit|`npm-cooldown-audit.yaml`|Report lockfile entries that do not satisfy the `.npmrc` supply-chain cooldown (never blocks)|
|Go Cooldown|`go-cooldown.yaml`|Gate a PR that adds or upgrades a direct Go module published inside the cooldown window|
|Tool Cooldown|`tool-cooldown.yaml`|Gate a PR that pins a CLI tool version — declared in `mise.toml` or `python/*.in` — published inside the cooldown window|
|Config Scan|`trivy-config.yaml`|Trivy misconfiguration scan of the Dockerfiles, gating at HIGH|
|SAST|`sast.yaml`|Opengrep (Semgrep-compatible) scan of first-party Go and TypeScript source with taint tracking|
|Lockfile Integrity|`lockfile-integrity.yaml`|Verify every npm `resolved` URL points at the official registry over HTTPS|
|OpenAPI Security|`openapi-security.yaml`|Spectral with the OWASP API Security ruleset over the OpenAPI definition|
|Fuzz|`fuzz.yaml`|Go native fuzzing over the parsers that accept external text|
|DAST|`dast.yaml`|OWASP ZAP API scan, driven by the OpenAPI definition, against the application booted inside the runner (report-only sample; see [DAST](#dast)) <!-- dast:line -->|
|Capability Diff|`capability-diff.yaml`|capslock report of capability changes in the Go dependency graph (report-only)|
|Notify|`notify.yaml`|Reusable `workflow_call` target that pushes a scheduled failure, or a finding from a non-blocking scanner, to a human|

Every scanner writes SARIF to GitHub code scanning where it can, and reports a finding on the PR through the shared `upsert-pr-comment` action (see [Result Comments](#result-comments) for when a comment is written at all).

#### Security Trigger Matrix

Each tool runs where its findings can actually change: a PR surfaces the risk the change itself introduces, a push to a protected branch keeps a code-scanning baseline for branch protection to judge, and a weekly schedule only exists for tools whose result can change while the code stands still (newly disclosed CVEs, new queries).

| Tool | Pull request | Push to protected branch | Schedule |
| --- | --- | --- | --- |
| gitleaks | all PRs | — | weekly, full history |
| Trivy secret | all PRs, working tree | — | weekly |
| TruffleHog | all PRs, diff only | — | weekly, full history |
| zizmor | when Actions files change | `develop` / `staging` / `production` / `release/*` | weekly (online audits) |
| Dependency Review | dependency-change PRs | — | — |
| govulncheck | Go / dependency-change PRs | same as above | weekly |
| Trivy FS | Go / dependency-change PRs | same as above | weekly |
| OSV-Scanner | dependency-change PRs | same as above | weekly |
| CodeQL | Go / TypeScript / Actions-definition-change PRs | same as above | weekly |
| OpenSSF Scorecard | — | default branch only | weekly |
| Image Scan | PRs into a deploy branch | — | weekly |
| Release gates (Trivy FS / OSV) | PRs into a deploy branch | — | — |
| npm cooldown audit | lockfile / `.npmrc` changes | same as above | weekly |
| Trivy config (misconfig) | Dockerfile-change PRs | same as above | — |
| Trivy licence | same trigger as Trivy FS | same as above | weekly |
| OSV diff | dependency-change PRs | — | — |
| Opengrep (SAST) | Go / TypeScript / dependency / spec-change PRs | same as above | weekly |
| lockfile-lint | lockfile-change PRs | — | — |
| Spectral (OpenAPI) | spec-change PRs | `release/*` / deploy branches | — |
| capslock | `go.mod`-change PRs | — | — |
| Go fuzzing | — | — | weekly |
| OWASP ZAP (DAST) | — | `develop` / `staging` / `production` / `release/*` | weekly <!-- dast:line --> |

Weekly runs are staggered across Monday, one scanner per hour, so a single hour does not queue every scanner at once: `0 0` Trivy FS, `0 1` govulncheck, `0 2` TruffleHog, `0 3` OSV-Scanner, `0 4` Scorecard, `0 5` CodeQL, `0 6` Image Scan, `0 7` gitleaks (full-history), `0 8` zizmor (online audits), `0 9` npm cooldown audit, `0 10` Opengrep, `0 11` fuzz.

DAST takes the next slot, `0 12`. It sits at the end of the rotation because it is the only one that builds and boots the application before it scans, so it is the longest and the least useful to have queued ahead of anything else. <!-- dast:line -->

Every scanner with a weekly schedule calls `notify.yaml` when its job ends in `failure` or `cancelled`. A PR failure is already visible to its author; a scheduled failure is visible to nobody, which is the case the notification exists for. `cancelled` is included because a job killed by a timeout or a runner fault reports that rather than `failure`.

Failure is not the only thing worth pushing. A report-only scanner leaves its job green on a finding, so failure mode can never fire for one; those call `notify.yaml` in detection mode instead, which names the actor, ref, commit and the findings themselves. Both modes skip delivery and leave the run green when no webhook secret is configured, so a fork is never failed by a notification it cannot send.

Which trigger a detection notification fires on follows from who the right recipient is. For the vulnerability scanners it is the scheduled run only — on a PR the finding is already in a comment addressed to the author, who introduced the dependency, whereas a weekly finding is a newly published advisory against code that stood still and reaches nobody. The npm cooldown audit is the exception and fires on every trigger, because the decision to bypass the cooldown belongs to a tech lead / architect who is not necessarily on the PR.

| Workflow | Fires when | Trigger |
| --- | --- | --- |
| `npm-cooldown-audit.yaml` | any cooldown finding | all |
| `trivy-fs.yaml` | fixable CRITICAL / HIGH / MEDIUM found | schedule |
| `vulnerability-check.yaml` | reachable vulnerability found | schedule |
| `osv-scanner.yaml` | promotion-blocking finding | schedule |

The other scheduled scanners need no detection notification: gitleaks, Trivy secret, TruffleHog, Opengrep, zizmor (at high), the image-scan gate and fuzzing all fail their job on a finding, so failure mode already delivers it. Three are deliberately left unconnected: the Trivy licence inventory reports licences nobody has yet agreed are problems (the same reason it writes no SARIF), while CodeQL and Scorecard publish to the code-scanning dashboard and expose no finding count to the workflow — a Scorecard "score dropped" notification would additionally need the previous score kept somewhere, which nothing here does.

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

#### Go Cooldown

Go has no counterpart to `min-release-age`: nothing lets `go get` refuse a version for being too new. That inverts the relationship between tool and guard. The npm audit above looks for evidence that an existing guard was bypassed, so reporting is enough; here the check **is** the guard, and reporting alone would leave the window existing nowhere.

`go-cooldown.yaml` therefore gates on a pull request, and only over the requirements the change adds or upgrades — everything already in `go.mod` is grandfathered, so the window applies going forward instead of holding every branch hostage to the state it inherited. Only **direct** requirements fail it. An indirect version is chosen by MVS and can sit above a direct dependency's own lower bound, where lowering it is not something the pull request can do; failing there would produce a red with no remedy, so those are reported.

The window is **7 days**, and the number comes from this repository rather than from npm. Go modules carry no install script — `go mod download` executes nothing — so the class where a freshly published version takes the machine at install time does not exist; what the window buys is time before malicious code is built and shipped. Measured against the history here, 7 days would have stopped 12 of the 47 commits that touched `go.mod` and 14 days only 3 more, so there is no cliff between them, and the one commit that already declared it was picking versions that "satisfy the cooldown" had waited 7.4 days.

Urgent overrides live in [`go-cooldown-bypass.toml`](../go-cooldown-bypass.toml), and every entry carries a deadline. An expired deadline, one reaching further than three months out, or an entry matching nothing in `go.mod` fails the check — and an invalid entry also stops working, so a lapsed bypass cannot quietly keep letting its module through. A deadline arrives without `go.mod` changing, which is why the schedule exists: the pull-request trigger alone would never see one expire.

**It never fails the build**, and that is a design decision rather than a default. Overriding the cooldown is a tech-lead / architect call — reacting to a CRITICAL advisory is the case it exists for — so a hard gate would block precisely the legitimate use. The non-blocking property lives in the tool itself, not in workflow configuration, so it cannot be turned into a gate by editing YAML.

Its scope is honest but narrow: **policy drift** — accidents, convention rot, a change in npm's own behaviour. It is not a defence against someone with commit access, who can delete the workflow in the same change. What it provides there is detection and attribution, with deterrence left to the organisation. The enforcement half is [`CODEOWNERS`](../CODEOWNERS), which reserves review of `**/.npmrc`, `**/package-lock.json`, `**/pnpm-lock.yaml`, `**/pnpm-workspace.yaml`, and the pin lockfiles to the owning role.

Only `mock-auth-server/` still resolves with npm. `scripts/` and `docs-viewer/` resolve with pnpm, whose `minimumReleaseAge` refuses a too-new version at resolution time instead of recording it and warning later (`minimumReleaseAgeStrict` makes that a hard failure rather than a silently widened window). There is no audit tool for that half because there is nothing to audit after the fact — which puts the whole weight on review of `pnpm-workspace.yaml` itself, hence its CODEOWNERS entry.

A pull request is audited against its base, so a finding names exactly the entries that change introduces and the PR comment persists as the record even after those versions age out of the window. The scheduled run audits every entry as a second net.

<!-- dast:begin -->
#### DAST

`dast.yaml` is the only workflow here that scans a *running* application. Every other security check reads files; this one builds the server, boots it against a seeded Postgres, and drives HTTP at it from OWASP ZAP, with the endpoint list taken from the bundled OpenAPI definition.

That shape is what decided the tool. Of the six DAST products in GitHub's code-scanning template catalogue, four run the scan on the vendor's own infrastructure — which cannot reach an API that exists only inside a GitHub-hosted runner — and the two that do run in the runner both require a paid token. ZAP needs no credential and scans from inside the job, so it is the only one that can see an ephemeral target at all.

**The scan runs authenticated, and that is the part most easily broken.** An unauthenticated scan collects 401 from every protected operation and stops at the surface, which looks like a completed scan and is not one. The `ci` environment profile wires the dev-only stub authenticator described in [`docs/design/auth.md`](../../docs/design/auth.md), so the credential is a `debug:<subject>` bearer token naming a seeded identity, and the job asserts that the credential is accepted before ZAP starts. Losing that assertion would not turn the check red — it would quietly shrink what the scan covers.

**It is report-only by design, not by omission.** The alert thresholds in [`.github/zap/rules.tsv`](../zap/rules.tsv) are tuned to this repository's sample API; gating a merge on sample rules would fail pull requests over findings the rules were never calibrated for. Alerts go to code scanning under the `zap-dast` category and to an artifact, and only a scan that could not run fails the job. ZAP emits no SARIF of its own, so the JSON report is mapped to it in the workflow; every alert is anchored to the OpenAPI bundle, because that file is what describes the surface the finding is about and pointing at a file that exists is what makes the alert navigable.

The whole thing is a working sample that a fork is expected to adjust or remove — `scripts/setup/remove-dast-setting` takes all of it in one pass, per [Phase 17 of the setup guide](../../docs/get-started/setup-repository.md).
<!-- dast:end -->

#### Runner Hardening

Every job in this directory starts with `step-security/harden-runner` in `egress-policy: block` mode with its own `allowed-endpoints`. It resolves every outbound connection against that list and refuses the rest, so a compromised action or transitive tool download cannot exfiltrate to an endpoint the job has no business reaching. File-integrity events are still recorded alongside.

The step stays **inline in every job**. Factoring it into a composite action would read as removing duplication, but the duplicated part is exactly the part that must not be shared: a single allowlist wide enough for every job is the union of all of them, which is no allowlist at all. What each job may reach is a property of that job.

Each list is the base set below plus what the job demonstrably does:

| Set | Endpoints | Applies to |
| --- | --- | --- |
| Base | harden-runner's own agent, the GitHub API / web / codeload hosts, `objects` / `raw` / `release-assets.githubusercontent.com`, `*.actions.githubusercontent.com`, `*.blob.core.windows.net` | every job — checkout, action download, artifact upload |
| Go | `proxy.golang.org`, `sum.golang.org`, `index.golang.org`, `storage.googleapis.com`, `dl.google.com` | `setup-go`, any `go build` / `go test` / `go run` |
| mise | `mise.jdx.dev`, `mise-versions.jdx.dev`, `aquaproj.github.io`, plus the Go and Sigstore sets | every `mise install`. aqua-backed tools resolve through GitHub releases (already in the base set), but mise then verifies their GitHub artifact attestations through Sigstore, and `go:`-backed tools resolve through the module proxy — so neither set is optional here |
| Node | `nodejs.org`, `registry.npmjs.org`, `get.pnpm.io` | `npm ci` / `pnpm install` |
| Python | `astral.sh`, `pypi.org`, `files.pythonhosted.org` | `uv`-resolved tooling (`sql-lint.yaml`) |
| Registry | the Docker Hub hosts, `mirror.gcr.io`, `ghcr.io`, `pkg-containers.githubusercontent.com` | image build / push, service containers, Trivy's DB and checks bundle |
| Scanner data | `semgrep.dev` (Opengrep rulesets), `api.osv.dev` / `api.deps.dev` (OSV), `vuln.go.dev` (govulncheck), the Scorecard data sources | the scanner that reads them |
| ZAP | `zaproxy.org` and its subdomains, plus the Registry set for the scanner image | `dast.yaml` (ZAP resolves its add-on manifest at startup) <!-- dast:line --> |
| Sigstore | `fulcio` / `rekor` / `tuf-repo-cdn` / `oauth2.sigstore.dev` | `deploy-app.yaml` (cosign keyless signing, attestation) **and every `mise install`** — mise fetches the Sigstore TUF root to verify artifact attestations, so a job that installs a tool but cannot reach `tuf-repo-cdn.sigstore.dev` fails before it runs anything |

The lists are inferred from what each job does rather than measured from audit data, so the first runs are expected to surface gaps. A blocked endpoint appears in the harden-runner run summary as a denied connection — that is the thing to read when a job fails for no reason visible in its own logs, and the fix is to widen that job's list, never to drop it back to `audit`.

**One job's failure mode is inverted and is called out where it lives.** `trufflehog.yaml` verifies a candidate credential by calling the service that issued it, and that set of services is open-ended. A missing endpoint there does not turn the job red; it turns a real leak into an unverified result the workflow does not report. Treat a disappearing TruffleHog finding as a possible allowlist gap.

Two jobs carry an assumption worth restating when forking: `deploy-app.yaml`'s build job assumes `ghcr.io` and the public Sigstore instance, and its deploy job is a placeholder whose list is the base set only — wiring a real deployment in means adding that cloud's control-plane hosts, since the OIDC exchange is an outbound call like any other.

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
|`upsert-pr-comment`|Marker-based PR comment upsert (detect existing → update / create) with a shared Commit / UpdatedAt footer, used by the result-commenting workflows; `status: success` updates an existing comment but creates none|
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
- **A fence around attacker-controlled text is sized from that text, never fixed.** `upsert-pr-comment` computes its fence as one backtick longer than the longest run in the body, but only on the `details-summary` path; without that input the body is passed through untouched, because several callers write Markdown — headings, tables, their own `<details>` — that is meant to render. A caller on that path which fences part of its body itself therefore owns the fence, and a fixed three-backtick one is closable by any body that reproduces source lines: a linter quoting a file the pull-request author wrote lands three backticks inside the block, and everything after renders as live Markdown under the bot's name. `sql-lint.yaml` builds its fences this way and sizes each from its own log; `capability-diff.yaml` leaves fencing to the action and wraps only its step summary. Callers must also keep each body under the action's `max-length`, which is applied *before* fencing — a body trimmed there loses its closing fence. `make actions-comment-fence-lint` covers the three parts that are mechanically decidable — no literal fence is emitted from a `run:` block, the duplicated `fence_for` helpers stay identical, and no pass-through workflow interpolates a value into an inline code span — but whether a body is attacker-controlled is not, so the rule is what holds.
- **The same rule covers inline code spans, which are just fences of length one.** A single backtick in the interpolated value closes the span, and the rest of the line reverts to live Markdown. A path is the case that bites: the only bytes it cannot hold are NUL and `/`, so backticks, `@`, and link syntax are all available, and a filename lifted straight out of `git diff --name-only` reaches the comment unaltered — `core.quotePath` escapes non-ASCII and control characters, not these. A pass-through caller therefore puts no repository-derived path in a span and none in bare Markdown either; it fences the whole list at a length taken from the list, exactly as above. The four `gen-*-artifacts-check.yaml` workflows and `sync-versions-check.yaml` emit their file lists that way. Where a body must keep rendering, fencing all of it is not available: `image-scan.yaml` opens its SBOM summary with a heading and a bold label, because that body is an inventory a reviewer reads rather than a log. There the template stays literal Markdown and every value is sized from itself instead — a scalar goes into a span one backtick longer than its own longest run, padded with the space CommonMark strips back off so that a value may begin or end with a backtick; a list is fenced from the list exactly as above; and a digest is matched against `^[0-9a-f]{64}$` and dropped to `unknown` when it does not fit. An SBOM string, unlike a path, is bounded by nothing, so a long run is elided and each value capped before the fence is sized — and the cap is what the whole scheme rests on, because the `max-length` trim that costs a fence its closing line costs a span its closing delimiter just the same, and an unclosed span hands the rest of the body back to live Markdown. Every contributor to that body is therefore bounded so their sum stays inside the cap. The linter keeps a file-scoped exclusion mechanism for a body not yet on one of these paths, on the same terms as `.github/zizmor.yml` — an entry names the issue tracking it and goes away when that finding is fixed, never a permanent allowlist. The check cannot see a span built through a variable or assembled by `jq`, so here too the rule outruns the linter. It reads the `details-summary` *value*, not just the key, because the action falls back to passing the body through when that input is empty — so `details-summary` must be a static non-empty string; an expression that could evaluate to empty is treated as pass-through and gets checked.
- Exceptions to zizmor's audits live in `.github/zizmor.yml`. `ignore` there is file-scoped on purpose, so a new workflow hitting the same audit still fails; entries are removed as the underlying finding is fixed rather than kept as a permanent allowlist.
- **An expression interpolated into a `run:` body is code, and only zizmor sees that.** `${{ }}` is substituted before the shell parses anything, so an unquoted `github.event.*` value ends the command and starts the attacker's. The shellcheck-based gates are structurally blind to this — see the `actions-shellcheck/` row in [`scripts/README.md`](../../scripts/README.md) for why. zizmor's `template-injection` audit judges the interpolation site instead and grades it by whether the expression's origin is attacker-controllable, which is why `make actions-zizmor` sits on the pre-commit hook beside `make actions-lint` rather than inside it. Bind the expression to an `env:` entry and read `"$VAR"` in the shell, where the value arrives as data.
- **Cache safety.** Caches are branch-scoped — a run restores only its own ref's cache plus the default branch's — so a pull-request run cannot write a cache that a later `release/*` push restores. That is why caching stays enabled on ordinary CI workflows. Poisoning becomes possible in two places. First, when untrusted PR code executes in a trusted scope while a cache is saved: `pull_request_target` and `workflow_run` run in the base ref's scope, so a workflow that checks out the PR head there would leave its cache exactly where privileged runs read it. Never combine those two; a workflow that must handle untrusted code keeps caching off. Second, **between workflows that share a branch scope but not a privilege level** — several ordinary workflows also run on pushes to protected branches, so a compromised one could leave a poisoned tool cache for a job holding `security-events: write` to restore and execute. Every job with that permission therefore sets `cache: false`, trading a slower install for not inheriting an artifact a lower-privileged run could have written.
- The `Detect changes` step in `auto-generate-docs.yaml` excludes coverage HTML and SchemaSpy timestamp churn so cosmetic regenerations do not open noise PRs.
- GitHub disables scheduled workflows automatically after 60 days without a commit, and it does so silently. Keeping them alive is out of scope for this template — no keepalive job is provided — so a fork that goes quiet should expect to re-enable them from the Actions tab.
- A repository created from a fork or template starts with every workflow in `disabled_fork` state, where nothing runs at all. `make enable-workflows` enumerates and enables them; it is idempotent and safe to re-run.
- **`claude.yaml` authorization.** Who may invoke Claude is decided by the action's own write-permission check, not by an allowlist in the workflow. Both alternatives break under forking: a hardcoded list of accounts locks a fork owner out of their own repository, and a repository variable holding that list is never inherited by a fork, so it resolves empty and nobody can invoke anything. A permission check resolves against whichever repository the workflow runs in and therefore needs no configuration to be correct anywhere. Two inputs would undo this and are deliberately left unset: `allowed_non_write_users` bypasses the check outright, and `allowed_bots` admits Apps that need neither installation nor write access. The workflow's own `if:` reads only the `github` context, so no comment starts a runner; it grants no authority. Restricting *who* cannot address prompt injection carried in a fork pull request — the invoker is trusted, the diff Claude reads is not — which is why `contents` stays read-only.
- `.spectral.yaml` and `.trivyignore.yaml` follow the same policy as `.github/zizmor.yml`: nothing is disabled in bulk, every entry carries the ADR or implementation that justifies it, and suppressions are scoped to a path (or a JSON pointer) so a new file hitting the same rule still fails.
- `fuzz.yaml` is scheduled rather than run per PR: a fuzz run explores a random corpus, so gating a merge on it would make the verdict depend on a coin flip. Crash reproducers are committed under `testdata/fuzz/` and replay as ordinary regression tests.
