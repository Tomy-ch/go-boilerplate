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

Three callers pass a constant `report`, all in `image-scan.yaml`: an SBOM inventory and the two Trivy tables are not verdicts, so none of them has a state that means "nothing to say", and that job only runs for a pull request into a deploy branch, where the contents of the image are what is under review.

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
| `fuzz.yaml`, `scorecard.yaml`, `notify.yaml`, `osv-release-gate.yaml`, `checkov.yaml` | 15 | no recent completed run to measure |
| `zap-api-scan.yaml` `dast` | 30 | no completed run to measure, and the job builds and boots the application before a scan whose length is set by the size of the OpenAPI definition |
| `code-ql.yaml` `codeql` | 30 | the limit covers whichever matrix leg is slowest, and no leg but `go` has a completed run to measure; `security-extended` is also a larger suite than the one the previous value was measured against |
| `secret-scan.yaml`, `trufflehog.yaml` | 15 | measured on pull requests only, where they scan a diff; the weekly run walks the full history and has never completed one to measure |
| `bearer.yaml` `bearer` | 20 | no completed run to measure, and the scan builds a data-flow model of the whole first-party tree before it reports anything |
| `sonarqube.yaml` `sonarqube` | 40 | no completed run yet measures the added Go and TypeScript coverage generation; the provisional limit reserves time for that work and Sonar's 10-minute analysis wait, then must be re-derived from the first completed run |
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
|Scripts Check|`scripts-check.yaml`|Type-check the repository's TypeScript helper scripts (`scripts/**/*.ts`), run the unit tests covering their decision logic, and run the 1:1 test-mapping gate, which also walks `docs-viewer/src/**` and `mock-auth-server/src/**`|
|OpenAPI Lint|`oapi-lint.yaml`|`redocly lint` the OpenAPI definition (naming / casing / descriptions / unused components)|
|App Boot Check|`app-di-startup-check.yaml`|Verify the application server starts successfully with DB|
|Job Boot Check|`job-boot-check.yaml`|Verify the job entrypoint boots and rejects an unknown job|
|Worker Boot Check|`worker-boot-check.yaml`|Verify the worker entrypoint boots (DI / DB) and rejects an unknown worker|
|Dockerfile Lint|`docker-lint.yaml`|Run hadolint on Dockerfiles (via go_tool_runner)|
|Markdown Lint|`md-lint.yaml`|Lint Markdown shape with markdownlint, validate every ` ```mermaid ` fence with the real parser, and check the `.claude/**` skill / agent definitions against reality and their `.codex/**` counterparts|
|Commitlint|`commitlint.yaml`|Lint every commit message the PR adds to the base branch — the route the `commit-msg` hook cannot cover|
|Pin Actions Check|`pin-actions-check.yaml`|Verify GitHub Actions are pinned to a SHA (supply-chain hardening)|
|Pin Images Check|`pin-images-check.yaml`|Verify Docker base images are pinned to a digest per the lockfile (supply-chain hardening)|
|Egress Check|`egress-check.yaml`|Verify every job's inline `allowed-endpoints` matches the SSOT (see [Runner Hardening](#runner-hardening))|

### Security

|Workflow|File|Description|
|---|---|---|
|CodeQL Scan|`code-ql.yaml`|CodeQL analysis on the `security-extended` suite, one matrix leg per language: `go`, `javascript-typescript` (mock-auth-server / docs-viewer / scripts) and `actions` (the workflow definitions themselves)|
|Dependency Scan|`trivy-fs.yaml`|Trivy filesystem scan for library vulnerabilities (developer-facing)|
|Release Dependency Scan|`trivy-release-gate.yaml`|Trivy filesystem scan on PRs into develop/staging/production|
|Grype Scan|`grype.yaml`|Anchore Grype filesystem scan of the same dependency manifests Trivy reads, against a different vulnerability database and a different matcher|
|Image Scan|`image-scan.yaml`|Build image, generate the SBOM in both SPDX-JSON and CycloneDX-JSON, run Trivy scan, check the built image against Dockle's practice rules, and re-check the CycloneDX SBOM with `trivy sbom`|
|Vulnerability Scan|`vulnerability-check.yaml`|govulncheck for actionable Go vulnerabilities|
|OSV Scan|`osv-scanner.yaml`|OSV database scan across the Go module graph and the npm lockfiles|
|Release OSV Scan|`osv-release-gate.yaml`|OSV scan on PRs into develop/staging/production, failing on HIGH or above|
|Secret Scan|`secret-scan.yaml`|Two independent scans of the working tree for committed secrets: gitleaks (wide regex / entropy net) and Trivy (curated rules, far fewer false positives), as separate jobs with separate verdicts|
|Secret Scan (TruffleHog)|`trufflehog.yaml`|TruffleHog scan for *verified* secrets — credentials that are actually live|
|Actions Static Analysis|`zizmor.yaml`|zizmor audit of the workflow / composite-action definitions themselves (same `make` gate as the pre-commit hook)|
|Dependency Review|`dependency-review.yaml`|Block a PR that introduces a newly vulnerable dependency|
|OpenSSF Scorecard|`scorecard.yaml`|Score the repository's security posture and publish the result|
|Go Cooldown|`go-cooldown.yaml`|Gate a PR that adds or upgrades a direct Go module published inside the cooldown window|
|Tool Cooldown|`tool-cooldown.yaml`|Gate a PR that pins a CLI tool version — declared in `mise.toml` or `python/*.in` — published inside the cooldown window|
|Config Scan|`trivy-config.yaml`|Trivy misconfiguration scan of the Dockerfiles, gating at HIGH|
|Checkov Scan|`checkov.yaml`|Checkov policy scan of the workflow definitions and the Dockerfiles, against a rule set neither zizmor nor Trivy ships (report-only)|
|SAST|`opengrep.yaml`|Opengrep (Semgrep-compatible) scan of first-party Go and TypeScript source with taint tracking|
|DevSkim Scan|`devskim.yaml`|DevSkim regex scan over every file in the tree, whatever its language|
|Bearer Scan|`bearer.yaml`|Bearer data-flow scan for sensitive values reaching a sink (report-only; Elastic License 2.0, see [Bearer's licence and removal](#bearers-licence-and-removal))|
|ESLint Scan|`eslint.yaml`|ESLint with `eslint-plugin-security` over the three TypeScript workspaces, one matrix leg each (report-only)|
|SonarQube Cloud Scan|`sonarqube.yaml`|SonarQube Cloud analysis of first-party source, read back over the Web API and converted to SARIF (**gates on Sonar's quality gate**, issue list report-only; needs `SONAR_TOKEN`, see [Removing the credential-bearing scanners](#removing-the-credential-bearing-scanners))|
|Lockfile Integrity|`lockfile-integrity.yaml`|Verify every npm `resolved` URL points at the official registry over HTTPS|
|OpenAPI Security|`openapi-security.yaml`|Spectral with the OWASP API Security ruleset over the OpenAPI definition|
|Fuzz|`fuzz.yaml`|Go native fuzzing over the parsers that accept external text|
|DAST|`zap-api-scan.yaml`|OWASP ZAP API scan, driven by the OpenAPI definition, against the application booted inside the runner (report-only sample; see [DAST](#dast))|
|Capability Diff|`capability-diff.yaml`|capslock report of capability changes in the Go dependency graph (report-only)|
|Agent Config Scan|`trustabl.yaml`|trustabl scan of the AI-agent configuration — subagent and skill declarations under `.claude/`, and MCP server declarations (report-only; see [Agent Config Scan](#agent-config-scan))|
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
| Trivy config (misconfig) | Dockerfile-change PRs | same as above | — |
| Checkov | Actions-definition / Dockerfile-change PRs | same as above | weekly |
| Dockle | PRs into a deploy branch | — | weekly (inside Image Scan) |
| Trivy SBOM | PRs into a deploy branch | — | weekly (inside Image Scan) |
| Trivy licence | same trigger as Trivy FS | same as above | weekly |
| OSV diff | dependency-change PRs | — | — |
| Opengrep (SAST) | Go / TypeScript / dependency / spec-change PRs | same as above | weekly |
| Grype | Go / dependency-change PRs | same as above | weekly |
| DevSkim | all PRs | `develop` / `staging` / `production` / `release/*` | weekly |
| Bearer | Go / TypeScript-change PRs | same as above | weekly |
| ESLint (security) | TypeScript-workspace-change PRs | same as above | weekly |
| SonarQube Cloud | Go / TypeScript / `sonar-project.properties`-change PRs | same as above | weekly |
| lockfile-lint | lockfile-change PRs | — | — |
| Spectral (OpenAPI) | spec-change PRs | `release/*` / deploy branches | — |
| capslock | `go.mod`-change PRs | — | — |
| Go fuzzing | — | — | weekly |
| OWASP ZAP (DAST) | when `zap-api-scan.yaml` or `.github/zap/**` changes | `develop` / `staging` / `production` / `release/*` | weekly |
| trustabl (agent config) | — | — | weekly |

Weekly runs are staggered across Monday, one scanner per hour, so a single hour does not queue every scanner at once: `0 0` Trivy FS, `0 1` govulncheck, `0 2` TruffleHog, `0 3` OSV-Scanner, `0 4` Scorecard, `0 5` CodeQL, `0 6` Image Scan, `0 7` gitleaks (full-history), `0 8` zizmor (online audits), `0 9` npm cooldown audit, `0 10` Opengrep, `0 11` fuzz.

DAST takes `0 12`. It is placed behind every file-reading scanner because it is the only one that builds and boots the application before it scans, so it is the longest and the least useful to have queued ahead of anything else.

The rotation then continues with `0 13` Grype, `0 14` DevSkim, `0 15` ESLint, `0 16` Bearer, `0 17` Checkov, `0 18` trustabl, `0 19` SonarQube Cloud.

The last one is the scanner whose analysis runs on a vendor's servers, and it is placed at the end for the same reason DAST is placed behind the file-reading scanners: its duration depends on a queue this repository does not control, so nothing useful is gained by having it queued ahead of a scanner that finishes on its own runner.

Every scanner with a weekly schedule calls `notify.yaml` when its job ends in `failure` or `cancelled`. A PR failure is already visible to its author; a scheduled failure is visible to nobody, which is the case the notification exists for. `cancelled` is included because a job killed by a timeout or a runner fault reports that rather than `failure`.

Failure is not the only thing worth pushing. A report-only scanner leaves its job green on a finding, so failure mode can never fire for one; those call `notify.yaml` in detection mode instead, which names the actor, ref, commit and the findings themselves. Both modes skip delivery and leave the run green when no webhook secret is configured, so a fork is never failed by a notification it cannot send.

Which trigger a detection notification fires on follows from who the right recipient is. For the vulnerability scanners it is the scheduled run only — on a PR the finding is already in a comment addressed to the author, who introduced the dependency, whereas a weekly finding is a newly published advisory against code that stood still and reaches nobody. The npm cooldown audit is the exception and fires on every trigger, because the decision to bypass the cooldown belongs to a tech lead / architect who is not necessarily on the PR.

| Workflow | Fires when | Trigger |
| --- | --- | --- |
| `trivy-fs.yaml` | fixable CRITICAL / HIGH / MEDIUM found | schedule |
| `vulnerability-check.yaml` | reachable vulnerability found | schedule |
| `osv-scanner.yaml` | promotion-blocking finding | schedule |
| `grype.yaml` | any vulnerability found | schedule |
| `devskim.yaml` | any finding | schedule |

The other scheduled scanners need no detection notification: gitleaks, Trivy secret, TruffleHog, Opengrep, zizmor (at high), the image-scan gate and fuzzing all fail their job on a finding, so failure mode already delivers it. Four are deliberately left unconnected: the Trivy licence inventory reports licences nobody has yet agreed are problems (the same reason it writes no SARIF), while CodeQL and Scorecard publish to the code-scanning dashboard and expose no finding count to the workflow — a Scorecard "score dropped" notification would additionally need the previous score kept somewhere, which nothing here does. Checkov joins them on the same terms: its baseline over this repository is twenty findings, most of them one rule reported once per workflow file. Dockle and `trivy sbom` need no wiring of their own: they run inside `image-scan.yaml`, whose scheduled failure already reaches a human.

ESLint, Bearer and trustabl are left unconnected for a different reason: their baselines are non-zero — over a hundred warnings for ESLint, fourteen findings for Bearer, and for trustabl one high finding per read-only subagent under `.claude/agents/` — so a detection notification keyed on "any finding" would fire every week regardless of what changed, which is the shape of a notification people learn to ignore. SonarQube Cloud joins them on that reason: it reports maintainability alongside security, so its baseline over an existing codebase is never zero.

#### Overlapping surfaces

Several tools can detect the same class of finding. **What must not be duplicated is the gate, not the tool**: one problem turning a pull request red twice means two places to suppress it and two places for a suppression to rot. Reporting is a different matter — a second engine reading the same files against its own database or rule set catches what the first has not learned about yet, and that redundancy is bought deliberately.

So a surface may carry several tools at once. What keeps the gate single is not that only one tool runs, but that **no two gating tools claim the same finding**. Two mechanisms hold that, and the table records both: where two tools would judge the same rule, one side is switched off for that surface (the third column names it, and why a capable tool is unused); where two gates coexist, they judge disjoint rule sets. `First-party Go source` is the case that shows the difference — Opengrep gates on Semgrep's ERROR band and `gosec` gates through golangci-lint, over the same files but never the same rule — so "one owner" there means one owner *per rule*, not one tool.

Every other tool on a shared surface is report-only, and the verdict on that surface belongs to the gate(s) marked below. A row with no `(gate)` marker gates nowhere: the dependency scanners report, and the blocking verdict is the release gates' (see [Release Gates](#release-gates)).

| Surface | Owner | Also capable, deliberately not used here |
| --- | --- | --- |
| Dockerfile security policy | `trivy-config.yaml` **(gate, HIGH+)** + `checkov.yaml` (Checkov, report-only) | Opengrep (its Dockerfile rules are excluded in `opengrep.yaml`) |
| Workflow definitions | `zizmor.yaml` (zizmor) **(gate)** + `code-ql.yaml` (`actions` leg) + `checkov.yaml` (Checkov, report-only) | — |
| Dockerfile style / correctness | `docker-lint.yaml` (hadolint) **(gate)** | — (a different layer, not a duplicate) |
| First-party Go source | `opengrep.yaml` (Opengrep, ERROR band) **(gate)** + `gosec` via `go-lint.yaml` **(gate)** — disjoint rule sets + `sonarqube.yaml` (SonarQube Cloud) **(gate, quality gate)** | — |
| OpenAPI conventions / naming | `oapi-lint.yaml` (redocly) **(gate)** | Spectral |
| OpenAPI security posture | `openapi-security.yaml` (Spectral) **(gate)** | redocly |
| Dependency vulnerabilities | `trivy-fs.yaml` (Trivy) + `osv-scanner.yaml` (OSV) + `grype.yaml` (Grype) — all report-only | — |
| First-party TypeScript source | `code-ql.yaml` (`javascript-typescript` leg) + `opengrep.yaml` (`p/typescript`) **(gate)** + `eslint.yaml` (`eslint-plugin-security`) + `sonarqube.yaml` (SonarQube Cloud) **(gate, quality gate)** | — |
| Any file, whatever its language | `devskim.yaml` (DevSkim) | — |
| AI-agent configuration (`.claude/**`, MCP declarations) | `trustabl.yaml` (trustabl) — report-only | — (no other scanner here parses a tool grant) |
| Sensitive values reaching a sink | `bearer.yaml` (Bearer) — report-only, over application code only (`/scripts` is excluded: repository tooling handles no user data, which is the whole of what this question asks) | — |
| Runtime image | `image-scan.yaml` (Trivy) **(gate)** + Dockle (practice rules, report-only) + `trivy sbom` (report-only — the same database as the gate, reading syft's package list instead of Trivy's own) | — |

The `First-party Go source` and `First-party TypeScript source` rows carry the vendor-hosted scanner as well. Sonar is the one deliberate departure from "one owner per rule" in this table. Its quality gate judges the analysis as a whole — new-code coverage and duplication alongside its own issue taxonomy — so it cannot be narrowed to the rules Opengrep and gosec do not claim, and a finding both engines recognize can turn a pull request red twice. That is accepted here because the alternative was discarding the vendor's verdict entirely, which left the scan reporting into a run that merged regardless.

#### Bearer's licence and removal

`bearer/bearer` is published under the **Elastic License 2.0**, which permits use, modification and redistribution but forbids providing the software to third parties as a hosted or managed service, and forbids circumventing its licence-key functionality. Running it inside this repository's own CI engages neither: nothing here is offered to a third party, and the CLI needs no key — its `--api-key` flag is documented as legacy and exists only for the vendor's discontinued cloud product.

Being outside the OSI definition is not what makes Bearer unusual here — CodeQL is not OSI-approved either, and gets no section of its own. What is worth writing down is that a repository created from this template inherits the workflow along with the licence, so a consumer who then wants to offer the tool as part of a service has a question to answer that the OSI-licensed scanners here do not raise. Removing Bearer is the answer, and it has to take all of this with it:

| Remove | Kept green by |
| --- | --- |
| `.github/workflows/bearer.yaml` | — |
| the `aqua:Bearer/bearer` line in [`mise.toml`](../../mise.toml) | `make tool-cooldown-gate` reads the pin from here |
| the `[job."bearer.yaml:bearer"]` section in [`.github/egress.toml`](../egress.toml) | `make egress-check` fails on a job section with no matching workflow |
| the `bearer.yaml` rows in this file and its `README.ja.md` translation — timeout table, Security table, trigger matrix, weekly rotation, overlapping surfaces — and this section | `make md-lint` checks the pair, not the rows |

`make pin-actions-check` needs nothing done to it as long as every action `bearer.yaml` used is still referenced elsewhere; it fails on a lockfile entry nothing references, so check that first if it goes red. The `level` defaulting in the summary step goes with the workflow: it exists because Bearer omits `level` from every result, and jq raises a runtime error on the sort key rather than falling through to `//`.

#### Removing the credential-bearing scanners

Two scanners here need something the repository cannot supply on its own. SonarQube Cloud needs a token for a vendor's service, and CodeQL needs GitHub Advanced Security, which is free for a public repository and billed for a private one. Both are free for this repository because it is public; a repository created from this template may be neither public nor willing to pay.

`make setup-remove-licensed-scanners` removes both in one run, and commits each product separately so a consumer who holds a licence for one of them can restore it with `git revert` on that commit alone. Which is why the removal is one script and not one per product: the decision a consumer actually makes once is "do I want scanners that bill me or phone a vendor", and the per-product choice is better expressed as an undo than as two scripts to remember.

The edits to this file and its translation are **not** in those per-product commits — they land in one final commit of their own. The products occupy adjacent rows of the same tables, so a per-product doc edit makes every `git revert` but the last one conflict here, which is the one thing the split exists to prevent. The cost is that a reverted scanner comes back working but undocumented; its rows can be read back out of that final commit.

`bearer.yaml` is deliberately **not** in that set. The Elastic License 2.0 costs nothing to run in CI and constrains only redistribution as a service, which is a different question with a different answer — see [Bearer's licence and removal](#bearers-licence-and-removal), which stays a manual procedure.

What the script takes with each product, and what has to survive:

| Removed with the product | Must survive |
| --- | --- |
| the workflow file, and `sonar-project.properties` for Sonar | — |
| the `[job."<workflow>:<job>"]` sections in [`.github/egress.toml`](../egress.toml) | — |
| the lockfile entries in [`.github/actions-pin.toml`](../actions-pin.toml) that nothing else references | `github/codeql-action@v4` — every other workflow that uploads SARIF references it |
| the rows and prose in this file and its `README.ja.md` translation | the rows of every scanner that stays |
| `.github/codeql/**` for CodeQL | — |

The lockfile rule is not a list of exceptions: the script counts references in the workflows that remain and deletes an entry only when the count reaches zero. `github/codeql-action@v4` is the case that shows why counting beats a list — it is registered with CodeQL, but every scanner that publishes SARIF calls `upload-sarif` from that same action, so removing CodeQL leaves the entry in place where a fixed list would have taken it. `actions/download-artifact@v7` is the opposite case: Sonar's report job is its only user today, so removing Sonar takes it along. `make pin-actions-check` and `make egress-check` both fail on an orphan, which is what turns a missed entry into a red run rather than a silent leftover.

The same counting is why reverting one scanner can leave `make pin-actions-check` red on an *unregistered* reference rather than an orphan: where two scanners in the removal set share an entry, it is deleted by whichever commit removed its last user, so restoring the earlier one brings back a `uses:` whose entry a later commit already took. The present set of two shares no such entry, so no revert hits this today — it returns the moment a third scanner joins. `make pin-actions-resolve` puts it back, and the check names the entry.

Registering `SONAR_TOKEN` stays a human step, as does creating the project on the vendor's side. Until it exists the leg skips itself and the run stays green — see [Result Comments](#result-comments) for why a missing credential is reported as a setup gap rather than as a scan result.

#### OSS scanners evaluated but not in the catalogue

GitHub's code-scanning starter catalogue is not the boundary of what could run here. Eight OSS tools outside it were evaluated together; four are now wired in and four were declined. The table below is the record, and it exists because **a repository created from this template is usually private, where the answers change** — a licence that costs nothing here may not be acceptable there, and a service that is free for a public repository may not be free for theirs. The reasoning is kept here rather than in the issue that produced it, because a consumer of this template can read this file and cannot read that issue.

Licences were read from each project's own licence file rather than from a third-party summary. Where a question could not be settled, the cell says so instead of guessing.

| Tool | Licence | Private / internal | Public | Verdict |
| --- | --- | --- | --- | --- |
| Dockle | Apache-2.0 | yes | yes | **Adopted** — inside `image-scan.yaml`. Practice rules over the built image, which no other scanner here reads. Runs entirely on the runner. Note that its last release is from January 2025, though the project still takes commits |
| `trivy sbom` | Apache-2.0 | yes | yes | **Adopted** — inside `image-scan.yaml`. Not a new tool but a subcommand of the Trivy already pinned here, so it raises no licence or supply-chain question of its own |
| Checkov | Apache-2.0 (CLI and Action alike) | yes | yes | **Adopted** — `checkov.yaml`. The CLI needs no account and reaches nothing outside the runner; the vendor's SaaS integration is a separate opt-in feature that is not used. Its `github_actions` rules are the part that earns its place, and they matter more, not less, once CodeQL is removed |
| KICS | core Apache-2.0, **Action GPL-3.0** | yes | yes | **Declined** — on distribution shape, not on licence. Its release archive ships the binary alone without the query library, and its aqua package is a `go_build` recipe mise cannot install, so no route to it stays inside this repository's version SSOT. Calling the Action from a workflow would create no GPL obligation, but it would leave the tool outside `mise.toml` and outside `tool-cooldown.yaml` |
| detect-secrets | Apache-2.0 | yes | yes | **Declined** — it would be the fourth secret engine after gitleaks, Trivy secret and TruffleHog |
| Renovate | **AGPL-3.0** (MIT through v11; AGPL from v12) | yes, self-hosted | yes, self-hosted | **Declined** — Dependabot plus the cooldown gates already cover this, and nothing here needs what Renovate adds. Running it unmodified for one's own dependency updates triggers no AGPL disclosure; the terms of Mend's hosted app were **not established** and would need checking by anyone who adopts that form |
| OpenSSF Allstar | Apache-2.0 | yes, with an org decision | yes | **Declined** — it enforces rather than detects, and it is an organisation-level GitHub App, so it is not something a template can hand down as a workflow file at all. It also wants authority over settings that [`.github/settings/branch-protection.json`](../settings/branch-protection.json) already owns. Adopting it means first deciding which of the two is authoritative, and granting an externally operated App read access to a private repository is the consuming organisation's call, not this template's |

Two limits on the table. It reports licence terms and whether an account is required; it does not reach any organisation's internal policy on GPL- or AGPL-licensed tooling, and where those disagree the organisation's policy governs. And a declined verdict is this repository's, decided against what already runs here — a consumer whose repository lacks these overlaps may well reach a different one, which is the reason the reasoning is written out rather than just the outcome.

#### DevSkim's version pin

Every other tool a workflow here installs is pinned in [`mise.toml`](../../mise.toml), which is what makes `tool-cooldown.yaml` able to gate a version published inside the supply-chain cooldown window. DevSkim is the exception: `microsoft/DevSkim` publishes no release binary and has no aqua package, so the only distribution is the NuGet global tool.

mise can in fact reach that — its `dotnet:` backend resolves NuGet packages — and it is still not used, for two reasons. The backend requires the .NET runtime to become a mise-managed tool as well, which puts a whole language runtime into the version SSOT for the sake of one linter. And [`scripts/tool-cooldown`](../../scripts/tool-cooldown) has no publish-time source for a `dotnet:` backend, so the entry would be reported as *unresolved* rather than gated. Moving the pin into `mise.toml` would therefore buy the appearance of coverage without the coverage.

The version consequently lives in `devskim.yaml`'s own `env:` block, and nothing guards it — bumping it means reading the release notes by hand.

The alternative — `microsoft/DevSkim-Action` — is worse on exactly this axis, not better. It is a Docker action whose `Dockerfile` builds from the floating `mcr.microsoft.com/dotnet/sdk:8.0` tag and then runs an unversioned `dotnet tool install`, so pinning the action to a SHA pins the recipe and not the code that ends up executing.

#### Release Gates

The dependency scanners are a double gate. On an ordinary PR they report only: a vulnerability inherited from the existing dependency tree is not something that PR introduced, and blocking there stops unrelated work while the update is prepared elsewhere. The blocking verdict happens on a PR into `develop` / `staging` / `production`, where the dependency state under review is the one about to be promoted.

| Gate | Fails on |
| --- | --- |
| `trivy-release-gate.yaml` | any Trivy finding, including one with no released fix |
| `osv-release-gate.yaml` | any OSV finding rated HIGH or CRITICAL, fixed or not, plus an unrated finding that has a fixed version |

Severity for the OSV gate comes from the advisory's own rating and falls back to the CVSS score osv-scanner aggregates per group. Advisories from the Go vulnerability database publish neither, so they cannot be measured against the HIGH threshold at all; those gate only when a fixed version exists, which keeps an advisory that can be neither rated nor updated away from turning every promotion permanently red. Both gates deliberately carry no `paths` filter — a promotion PR often changes no manifest, and a required check has to run to be able to block.

#### Go Cooldown

Go has no counterpart to `min-release-age`: nothing lets `go get` refuse a version for being too new. That inverts the relationship between tool and guard: pnpm refuses a too-new version at resolution time, so the resolver is the guard; here the check **is** the guard, and reporting alone would leave the window existing nowhere.

`go-cooldown.yaml` therefore gates on a pull request, and only over the requirements the change adds or upgrades — everything already in `go.mod` is grandfathered, so the window applies going forward instead of holding every branch hostage to the state it inherited. Only **direct** requirements fail it. An indirect version is chosen by MVS and can sit above a direct dependency's own lower bound, where lowering it is not something the pull request can do; failing there would produce a red with no remedy, so those are reported.

The window is **7 days**, and the number comes from this repository rather than from npm. Go modules carry no install script — `go mod download` executes nothing — so the class where a freshly published version takes the machine at install time does not exist; what the window buys is time before malicious code is built and shipped. Measured against the history here, 7 days would have stopped 12 of the 47 commits that touched `go.mod` and 14 days only 3 more, so there is no cliff between them, and the one commit that already declared it was picking versions that "satisfy the cooldown" had waited 7.4 days.

Urgent overrides live in [`go-cooldown-bypass.toml`](../go-cooldown-bypass.toml), and every entry carries a deadline. An expired deadline, one reaching further than three months out, or an entry matching nothing in `go.mod` fails the check — and an invalid entry also stops working, so a lapsed bypass cannot quietly keep letting its module through. A deadline arrives without `go.mod` changing, which is why the schedule exists: the pull-request trigger alone would never see one expire.

**It never fails the build**, and that is a design decision rather than a default. Overriding the cooldown is a tech-lead / architect call — reacting to a CRITICAL advisory is the case it exists for — so a hard gate would block precisely the legitimate use. The non-blocking property lives in the tool itself, not in workflow configuration, so it cannot be turned into a gate by editing YAML.

Its scope is honest but narrow: **policy drift** — accidents, convention rot, a change in the resolver's own behaviour. It is not a defence against someone with commit access, who can delete the workflow in the same change. What it provides there is detection and attribution, with deterrence left to the organisation. The enforcement half is [`CODEOWNERS`](../CODEOWNERS), which reserves review of `**/pnpm-lock.yaml`, `**/pnpm-workspace.yaml`, and the pin lockfiles to the owning role.

All three packages resolve with pnpm, whose `minimumReleaseAge` refuses a too-new version at resolution time instead of recording it and warning later (`minimumReleaseAgeStrict` makes that a hard failure rather than a silently widened window). There is no audit tool for it because there is nothing to audit after the fact — which puts the whole weight on review of `pnpm-workspace.yaml` itself, hence its CODEOWNERS entry.

A pull request is audited against its base, so a finding names exactly the entries that change introduces and the PR comment persists as the record even after those versions age out of the window. The scheduled run audits every entry as a second net.

#### DAST

`zap-api-scan.yaml` is the only workflow here that scans a *running* application. Every other security check reads files; this one builds the server, boots it against a seeded Postgres, and drives HTTP at it from OWASP ZAP, with the endpoint list taken from the bundled OpenAPI definition.

That shape is what decided the tool. Of the six DAST products in GitHub's code-scanning template catalogue, four run the scan on the vendor's own infrastructure — which cannot reach an API that exists only inside a GitHub-hosted runner — and the two that do run in the runner both require a paid token. ZAP needs no credential and scans from inside the job, so it is the only one that can see an ephemeral target at all.

**The scan runs authenticated, and that is the part most easily broken.** An unauthenticated scan collects 401 from every protected operation and stops at the surface, which looks like a completed scan and is not one. The job runs under the `dast` environment profile, which wires the real JWKS-backed authenticator described in [`docs/design/auth.md`](../../docs/design/auth.md) rather than the dev-only stub `ci` uses, so the credential is a JWT the mock auth server actually signed and the scan drives signature verification, `typ` checking and `kid` resolution on every request. The job asserts that the credential is accepted before ZAP starts. Losing that assertion would not turn the check red — it would quietly shrink what the scan covers.

**It is report-only by design, not by omission.** The alert thresholds in [`.github/zap/rules.tsv`](../zap/rules.tsv) are derived from what this API happens to answer today; gating a merge on them would fail pull requests over findings they were never calibrated for. Alerts go to code scanning under the `zap-dast` category and to an artifact, and only a scan that could not run fails the job. ZAP emits no SARIF of its own, so the JSON report is mapped to it in the workflow; every alert is anchored to the OpenAPI bundle, because that file is what describes the surface the finding is about and pointing at a file that exists is what makes the alert navigable.

The thresholds and the scanned surface are expected to be re-derived against the API they are pointed at — see [Phase 17 of the setup guide](../../docs/get-started/setup-repository.md).

#### Agent Config Scan

`trustabl.yaml` scans a surface no other check here reads: the AI-agent configuration itself. zizmor and Checkov read the workflow definitions, CodeQL and the Go linters read the source, and none of them parse a subagent's `tools:` grant or a skill's `allowed-tools:`. The rule packs that matter for this repository are the Claude subagent and skill ones; the engine also ships packs for the OpenAI, Google ADK, LangChain, CrewAI and MCP ecosystems, which find nothing here.

**Three separately-versioned artifacts run in that one step, and only the first is pinned by pinning the action.** The action downloads the engine binary from the vendor's releases at run time, and the engine clones its rule pack from a second repository — both defaulting to a moving target. Left at their defaults they would put unreviewed third-party code on the runner every week, outside the cooldown window every other pin in this repository observes. The workflow therefore names all three: the action by SHA through [`.github/actions-pin.toml`](../actions-pin.toml), the engine by release tag, and the rule pack by tag. The rule pack is the weak link — the engine resolves that input against branches and tags but never a commit, so a re-pointed tag would be adopted silently. The engine logs the SHA it actually cloned, which is where a re-point would show.

**It is report-only, and the baseline is the reason.** The subagent rules flag any grant of `Bash`, and every read-only reviewer under `.claude/agents/` holds one — the grant is what lets a reviewer run `git diff` or `go build`, so the finding describes the design rather than a defect in it. A severity gate would fail on that baseline from the first run. The useful half is the skill pack, which catches a bare `Bash` in a skill's `allowed-tools:` where the narrow `Bash(git status:*)` form was meant; `allowed-tools` is an auto-approval list rather than a sandbox, so the difference is real. Findings reach a human through the step summary and the `trustabl` artifact.

SARIF upload and the sticky PR comment are both switched off. Each costs a write scope — `security-events: write` and `pull-requests: write` respectively — handed to a third-party binary, and a report-only scanner has no claim on either. That also keeps the job at `contents: read`.

#### Runner Hardening

Every job in this directory starts with `step-security/harden-runner` in `egress-policy: block` mode with its own `allowed-endpoints`. It resolves every outbound connection against that list and refuses the rest, so a compromised action or transitive tool download cannot exfiltrate to an endpoint the job has no business reaching. File-integrity events are still recorded alongside.

The step stays **inline in every job**, and that is a constraint rather than a preference. A local composite action (`uses: ./.github/actions/*`) resolves only once the repository is checked out, and harden-runner has to run *before* the checkout — the checkout is itself an outbound call, and guarding it is the point. Factoring the step out would open the window it exists to close. Expect this proposal to recur; the answer is that it is not available, not that it was weighed and declined.

**What is not fixed is where the list comes from.** [`.github/egress.toml`](../egress.toml) is the source of truth. `make egress-apply` writes it into every job, and `make egress-check` fails when an inline block has drifted from it — on the pre-commit hook and in `egress-check.yaml`.

**A job declares its capability class, not its hosts.** What a job reaches follows from what it *does* — install tooling, build an image, boot a database — and not from the job's own identity: execution descends from `make` into docker into `mise` inside the container, so the endpoints a job needs are not visible in its YAML. Four classes cover it, and a job names the ones that apply:

| Class | Endpoints | Applies to |
| --- | --- | --- |
| `base` | harden-runner's own agent, the GitHub API / web / codeload hosts, `objects` / `raw` / `release-assets.githubusercontent.com`, `*.actions.githubusercontent.com`, `*.blob.core.windows.net` | **every job, implicitly** — checkout, action download, artifact upload. It is never written in `classes` |
| `mise` | mise's own distribution, plus every backend `mise.toml` resolves through: aqua / GitHub releases, the Go toolchain and module proxy, `downloads.sqlc.dev`, the npm registry and `get.pnpm.io`, `astral.sh` and PyPI — and Sigstore, because mise verifies each tool's GitHub artifact attestation through it | any job that installs tooling. A job that only runs `setup-go` names this class too: the module proxy lives here, and a narrower Go-only class would be one more classification to get wrong |
| `image` | the Docker Hub hosts and both CDNs, `mirror.gcr.io`, `ghcr.io`, `pkg-containers.githubusercontent.com`, and the Alpine / Debian package mirrors. Inherits `mise`, because the image build runs `mise install` inside the container | image build / push, service containers, Trivy's DB and checks bundle, and anything driving docker through `make` |
| `db` | the PGDG apt repository and the Ubuntu archive mirrors that the Postgres service container installs from | jobs that boot Postgres |

Anything genuinely particular to one job goes in that job's `extra`: a scanner's data source (`semgrep.dev`, `api.osv.dev`, `vuln.go.dev`), a deploy target, `hooks.slack.com` for the notifier, `zaproxy.org` for the DAST job (ZAP resolves its add-on manifest at startup). A host that turns up in a second job's `extra` belongs in a class instead.

**The classes are deliberately coarse.** Splitting them finer buys a tighter allowlist and costs more classification decisions, and a job in the wrong class is the failure this arrangement exists to remove. A class that is slightly wider than one job needs still refuses everything outside it; a job in the wrong class fails the build.

To add an endpoint: edit `.github/egress.toml` — into the class when the need follows from a capability, into the job's `extra` when it does not — then run `make egress-apply` and commit the generated blocks. Never hand-edit an inline block; `make egress-check` rejects it.

A blocked endpoint appears in the harden-runner run summary as a denied connection. That is the thing to read when a job fails for no reason visible in its own logs, and the fix is to widen the class or the `extra`, never to drop the job back to `audit`.

**One job's failure mode is inverted and is called out where it lives.** `trufflehog.yaml` verifies a candidate credential by calling the service that issued it, and that set of services is open-ended. A missing endpoint there does not turn the job red; it turns a real leak into an unverified result the workflow does not report. Treat a disappearing TruffleHog finding as a possible allowlist gap. It is the only job on `egress-policy: audit`, declared as such in the SSOT so that it carries no `allowed-endpoints` at all — and `make egress-check` fails if the two ever disagree.

Two jobs carry an assumption worth restating when forking: `deploy-app.yaml`'s build job assumes `ghcr.io` and the public Sigstore instance (its `extra`), and its deploy job is a placeholder that declares no class at all, so it gets `base` and nothing else — wiring a real deployment in means adding that cloud's control-plane hosts to its `extra`, since the OIDC exchange is an outbound call like any other.

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
