---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [ci, security, supply-chain]
---

# ADR-0080: Pin GitHub Actions by SHA with a supply-chain quarantine

## Status

accepted

## Context

GitHub Actions references of the form `uses: owner/repo@v1.2.3` resolve the tag at
workflow execution time. Tags are mutable: a compromised maintainer or a tag-hijacking
attack can change what `v1.2.3` points to after the tag was reviewed. This is a
well-documented supply-chain attack vector — a malicious actor who controls an action
repository can silently replace a tagged release with code that exfiltrates secrets from
the runner environment.

Pinning by commit SHA (`uses: owner/repo@<40-char-sha>`) makes the reference immutable:
the SHA is content-addressed and cannot be silently redirected. The cost is that SHA
strings are opaque to human reviewers, and there is no standard mechanism in YAML to
annotate them with the corresponding tag.

Additionally, even a legitimately published new release may be untrustworthy for a short
window after publication: the upstream repository may have been recently compromised and
the tag could point to a malicious commit that has not yet been detected by the community.
A quarantine period guards against adopting brand-new releases immediately.

The lock-in concern from [ADR-0001](0001-avoid-lock-in.md) applies here too: updating
pinned SHAs should require a conscious, auditable action rather than silent drift.

## Decision

Maintain a TOML lockfile (`.github/actions-pin.toml`) as the single source of truth for
all external GitHub Actions SHA pins. The lockfile maps each `owner/repo@tag` key to the
resolved 40-character commit SHA.

A Go tool (`scripts/pin-actions/main.go`) provides three subcommands:

- `resolve`: walks `.github/workflows/*.yaml` and `.github/actions/*/action.yaml`,
  collects all external `uses:` references, resolves each tag to a commit SHA via
  `git ls-remote`, and writes the result to `.github/actions-pin.toml`. For annotated
  tags, the dereferenced commit SHA (`^{}`) is used. Accepts a `--min-age-days` flag:
  when set, any SHA whose corresponding release (or commit) is younger than the given
  number of days is quarantined — if a previous pin exists it is retained, otherwise the
  reference is skipped entirely.
- `apply`: reads the lockfile and rewrites every `uses:` line in the workflow files to
  `uses: owner/repo@<sha> # <tag>`, preserving the tag in a trailing comment for human
  readability.
- `check`: performs the same rewrite logic in dry-run mode and exits non-zero if any
  workflow line is unpinned, stale, or absent from the lockfile. Used in CI and the
  pre-commit hook.

The pre-commit hook (`pin-actions` in `.lefthook.yaml`, glob-scoped to workflow YAML, action YAML, and
`actions-pin.toml`) runs `make pin-actions-check` on every commit that touches workflow
files, ensuring that un-pinned or stale references are blocked before they reach the
remote.

Already-pinned lines (`@<sha> # <tag>`) are idempotently handled: the tool reads the
tag from the comment and re-resolves it, so running `resolve` repeatedly is safe.

## Consequences

### Positive Consequences

- All external action references are immutable at execution time; a compromised upstream
  tag cannot silently change what runs in CI.
- The lockfile is the SSOT: updating a pin requires a deliberate `resolve` + `apply`
  cycle, producing a reviewable diff.
- The `--min-age-days` quarantine flag allows adopting a cautious stance toward newly
  published releases.
- Human reviewers can read the tag from the inline comment without needing to look up
  the SHA.
- The pre-commit check prevents unpinned references from being committed.

### Negative Consequences

- Updating any action requires running `make pin-actions-resolve && make pin-actions-apply`
  and committing the result; it cannot be done by simply changing a version string.
- The lockfile must be kept in sync with the workflow files; a workflow added without
  running `resolve` will fail the `check` step.
- The quarantine period means the team may not be able to immediately adopt a critical
  security fix in an upstream action if it was published too recently.

## Alternatives Considered

### Tag-based references without pinning

Simple, human-readable. Vulnerable to tag mutation attacks. Rejected on supply-chain
security grounds.

### Dependabot for Actions updates

Dependabot can open PRs to update SHA pins. Compatible with this approach but does not
enforce pins for new workflow additions or handle the quarantine requirement. Can be
layered on top of this mechanism as a complementary update automation.

### Third-party SHA-pinning tools (e.g. Renovate with `pinDigests`)

Functionally similar. Rejected in favour of a purpose-built in-repo tool to avoid
adding an external SaaS dependency on the CI configuration management path — consistent
with [ADR-0001](0001-avoid-lock-in.md).

## Notes

- Sources: `.github/actions-pin.toml` (lockfile SSOT),
  `scripts/pin-actions/main.go` (resolve / apply / check tool),
  `.github/workflows/` (`uses:` lines use the `@sha # tag` format).
- The pre-commit enforcement is defined in `.lefthook.yaml` (`pin-actions` command).
- Related: [ADR-0001](0001-avoid-lock-in.md) — the general lock-in avoidance principle
  that motivates keeping the tooling in-repo.
