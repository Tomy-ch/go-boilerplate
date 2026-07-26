---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [ci, security]
---

# ADR-0080: Multi-layer security scanning (reachability-filtered govulncheck + scheduled CodeQL SAST + secret + fs scans)

## Status

accepted

## Context

A Go web service has several distinct attack surfaces that require different scanning
approaches:

- **Known CVEs in dependencies**: Traditional dependency scanners report every CVE in
  every transitively imported module, including ones in code paths that the application
  never executes. This produces a high false-positive rate that erodes trust in the
  scanner.
- **Logic-level security bugs**: SQL injection, path traversal, insecure deserialization,
  and similar patterns cannot be detected by dependency version checks; they require
  semantic analysis of the source code.
- **Leaked secrets**: API keys, passwords, or private keys accidentally committed to the
  repository need a different detector (pattern-based rather than semantic).
- **Dependency vulnerability breadth**: A second scan that covers the full dependency
  tree (including non-reachable code) provides a broader safety net for prioritisation
  and advisory tracking.

Two further surfaces were added as the pipeline matured:

- **The workflows and their runners themselves**: a CI definition is code with credentials,
  and a compromised action or a template-injected `run:` block is an attack on the build
  rather than on the product.
- **The shape of the API contract**: an endpoint that declares no bound on its input, or no
  authentication, is a defect in the spec that no code scanner reads.

No single tool covers these surfaces adequately. A layered approach using purpose-fit tools
is required.

## Decision

Operate independent security scanning layers, each owning one surface with its own trigger,
threshold, and gate policy. The layers below are the current set; the organising principle
is one surface per tool, and one gate policy per workflow.

**Layer boundaries that matter when adding a tool:**

- A finding that the change under review *introduced* blocks immediately (misconfiguration,
  a newly added vulnerable dependency, a poisoned lockfile, a spec that loosens an input
  bound). A finding *inherited* from the existing tree is reported on the PR and gated at
  the promotion boundary instead — blocking there stops unrelated work while the fix is
  prepared elsewhere. This is why `trivy-fs.yaml` / `osv-scanner.yaml` report while
  `trivy-release-gate.yaml` / `osv-release-gate.yaml` block.
- Scanners are split by *surface × gate policy*, not by tool. The same tool appears in more
  than one workflow when the questions differ (Trivy scans dependencies, Dockerfiles, and
  licences in three workflows with three thresholds), and overlapping detections are
  resolved by giving the surface a single owner rather than gating twice on one finding.

**The layers:**

**1. Reachability-filtered vulnerability scan — govulncheck** (`vulnerability-check.yaml`):
Triggered on every PR that touches `.go`, `go.mod`, or `go.sum`. Runs
`govulncheck -json ./...` (JSON mode) and post-processes the output with `jq` to retain
only findings whose call trace has depth > 1 (i.e. the vulnerable symbol is actually
reachable from the application's call graph). Unreachable CVEs are excluded from the
actionable count. The reachable-finding count is surfaced as an upsert PR comment for
prioritisation. This scan is **advisory** — it reports reachable findings on the PR rather
than hard-failing the build on them (unlike the secret scan, which does fail closed).

**2. SAST — CodeQL** (`code-ql.yaml`, lines 21–22):
Triggered on PRs touching Go files, on pushes to `release/*`, `develop`, `staging`, and
`production` branches, and on a weekly schedule (`cron: '0 0 * * 1'`). Uses GitHub's
CodeQL engine with the Go language pack to perform semantic analysis and detect
logic-level vulnerabilities. Results are uploaded to GitHub Security as SARIF.

**3. Secret scan — gitleaks** (`secret-scan.yaml`):
Triggered on every PR without path restriction (secrets can appear in any file type).
Runs gitleaks via the Docker-based `go_tool_runner` and posts the result as an upsert
PR comment. Exits non-zero if gitleaks finds any match.

**4. Filesystem dependency scan — Trivy** (`trivy-fs.yaml`):
Triggered on PRs touching `.go`, `go.mod`, or `go.sum`, and on a weekly schedule
(`cron: '0 0 * * 1'`). Runs `trivy fs` in `vuln` scan mode for `library` vulnerabilities
at `CRITICAL`, `HIGH`, and `MEDIUM` severity, ignoring unfixed vulnerabilities. Produces
both a Markdown summary (posted as a PR comment) and a SARIF report uploaded to GitHub
Security via the CodeQL upload action. Reports only; the blocking verdict belongs to layer 5.

**5. Promotion gates — Trivy / OSV** (`trivy-release-gate.yaml`, `osv-release-gate.yaml`):
Run on PRs into `develop` / `staging` / `production`, where the dependency state under
review is the one about to ship. Unlike layer 4 these count unfixed findings too, because
at the promotion boundary a known-severe vulnerability is a decision rather than a backlog
item.

**6. Broad dependency scan — OSV-Scanner** (`osv-scanner.yaml`):
Covers the Go module graph and both npm lockfiles through one advisory source. Its
`osv-diff` job additionally blocks a PR that *introduces* an advisory the base branch did
not have, with no severity or fixability threshold — that gate asks only "did this change
add something new", which is the GHAS-independent equivalent of Dependency Review.

**7. Configuration scan — Trivy config** (`trivy-config.yaml`):
Dockerfile misconfiguration at `HIGH` and above, blocking. Accepted exceptions are pinned
per path in `.trivyignore.yaml`. hadolint (`docker-lint.yaml`) lints the same files for
style and correctness; this layer is the security-policy layer above it.

**8. Licence inventory — Trivy licence** (job in `trivy-fs.yaml`):
Report-only and permanently so until a prohibited-licence policy exists.

**9. SAST on first-party source — Opengrep** (`sast.yaml`):
Semgrep-compatible rules (`p/golang`, `p/owasp-top-ten`) with intra-file taint tracking,
scoped to `internal` / `pkg` / `cmd` / `database` / `openapi` and gating at `ERROR`.
Dockerfiles are deliberately excluded here because layer 7 owns them.

**10. Secret detection — TruffleHog** (`trufflehog.yaml`):
Complements layer 3 by reporting only *verified* secrets — credentials that are actually
live — where gitleaks covers the regex-based side.

**11. Workflow and runner integrity — zizmor + Harden Runner** (`zizmor.yaml`, and a step in
every job): static analysis of the workflow definitions themselves (template injection,
excessive permissions, credential persistence), plus egress auditing on every runner.

**12. Supply-chain provenance — Scorecard, Dependency Review, lockfile-lint, capslock**
(`scorecard.yaml`, `dependency-review.yaml`, `lockfile-integrity.yaml`,
`capability-diff.yaml`): repository posture scoring, GHAS-backed dependency review, lockfile
`resolved`-URL verification against the official registry, and a report-only capability
diff on the Go dependency graph.

**13. Fuzzing — Go native** (`fuzz.yaml`): weekly randomised input exploration over the
parsers that accept external text. It is the only layer that finds defects nobody thought
to write a test for; its first run found an unbounded-exponent hang in `pkg/decimal`.

**14. API contract security — Spectral** (`openapi-security.yaml`): the OWASP API Security
ruleset over the OpenAPI definition, asking about unbounded input, undeclared
authentication, and server inventory. Convention linting stays with redocly
([ADR-0010](0010-redocly-modular-spec-pipeline.md)); the two do not overlap.

Scheduled runs are staggered one per hour on Monday so a single hour does not queue every
scanner. Any scanner with a weekly schedule calls `notify-failure.yaml` on failure, because
a scheduled failure has no author watching it.

## Consequences

### Positive Consequences

- Govulncheck's reachability filter reduces noise: only vulnerabilities in code the
  application actually calls are flagged as blocking.
- CodeQL covers logic-level bugs that version-based scanners miss entirely.
- The weekly schedule on CodeQL and Trivy catches newly published CVEs against the
  current codebase even without code changes.
- Separate tools scanning separate surfaces mean a failure in one layer is
  independently actionable.

### Negative Consequences

- The workflow count is substantial, and each one is surface area to maintain. The
  split is deliberate (surface × gate policy), but it means adding a scanner is never
  just "add a tool" — it requires deciding which surface owns the finding.
- Detections overlap across tools, and the overlap has to be resolved by hand each time
  (Opengrep and Trivy both flag a root-user Dockerfile; Opengrep and gosec both flag
  `math/rand`). Left unresolved, the same finding gates twice and gets suppressed twice.
- Govulncheck's reachability analysis requires Go module awareness; if the module graph
  changes structure, filtering assumptions may need revisiting.
- Secret scan runs unconditionally on every PR (no path filter), adding a fixed overhead
  to all PR workflows.
- CodeQL's scheduled run on the weekly cron may produce findings that are not associated
  with any open PR, requiring triage by maintainers proactively.

## Alternatives Considered

### Single dependency scanner only (e.g. Dependabot)

Dependabot updates dependencies reactively but does not perform reachability analysis or
SAST. It would miss logic-level bugs entirely.

### govulncheck without reachability filtering

Running govulncheck without the `trace > 1` post-filter produces a large volume of
advisory-level findings for code paths never exercised. PR authors would habitually
dismiss all govulncheck output, destroying the signal.

### Consolidate all scanning into a single Trivy step

Trivy can scan for both CVEs and secrets, but its semantic analysis is not equivalent to
CodeQL, and its reachability awareness is weaker than govulncheck. Consolidation would
sacrifice depth for simplicity.

## Notes

- Sources: `.github/workflows/code-ql.yaml` lines 21–22,
  `.github/workflows/vulnerability-check.yaml` lines 57–61,
  `.github/workflows/secret-scan.yaml`,
  `.github/workflows/trivy-fs.yaml`.
- The `gosec` linter in `.golangci-full.yaml` provides an additional in-process security
  check during static analysis (see [ADR-0075](0075-two-layer-golangci-config.md)).
