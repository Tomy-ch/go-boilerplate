---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [ci, security]
---

# ADR-0076: Multi-layer security scanning (reachability-filtered govulncheck + scheduled CodeQL SAST + secret + fs scans)

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

No single tool covers all four surfaces adequately. A layered approach using purpose-fit
tools is required.

## Decision

Operate four independent security scanning layers, each with its own trigger schedule
and tool:

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
Security via the CodeQL upload action.

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

- Four separate workflows add CI complexity and surface area to maintain.
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
  check during static analysis (see [ADR-0071](0071-two-layer-golangci-config.md)).
