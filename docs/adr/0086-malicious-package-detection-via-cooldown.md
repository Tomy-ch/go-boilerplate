---
status: accepted
date: 2026-07-26
deciders: [maintainers]
tags: [ci, security, dependencies, setup-review]
---

# ADR-0086: Accept a publication cooldown as the primary defence against malicious packages, without a dedicated detector

## Status

accepted

## Context

Vulnerability scanners answer "is this dependency *known* to be vulnerable". A different
class of supply-chain attack defeats that question entirely: a package that is not
vulnerable but **malicious** — a typosquatted name, or a legitimate package whose
maintainer account or release pipeline was compromised. The artifact is published through
the normal channel with valid integrity hashes, so every lockfile, checksum, and CVE
database agrees it is fine.

This is not hypothetical for this repository's dependency graph. On 2026-07-14 four
`@asyncapi` packages were published from compromised release pipelines carrying a
credential-stealing payload; `@asyncapi/specs` is reachable from
`@stoplight/spectral-cli` → `@stoplight/spectral-rulesets`, which this repository uses for
OpenAPI security linting. What protected consumers was not a scanner — it was the window
between publication and discovery, and npm's eventual removal of the versions.

The pressure to add a dedicated detector is therefore real. The problem is that the tools
which would answer it do not exist in usable form:

- **npm** has no practical open-source malicious-package detector. The credible offerings
  (Socket, Snyk Advisor, Phylum) are commercial SaaS that require sending the dependency
  graph to a third party — which conflicts with ADR-0001 (avoid lock-in) and adds an
  external dependency to CI.
- **Go** has `capslock`, which reports the capabilities (network, exec, unsafe, …) a
  dependency can reach. It is genuinely useful, but it covers only one of the two
  ecosystems in this repository, and it reports *capability*, not *intent*: a logging
  library that legitimately gains file access is indistinguishable from one that gained it
  maliciously.

## Decision

We deliberately do **NOT** adopt a malicious-package detector. The primary mitigation is a
**publication cooldown**: a dependency version younger than the configured window is
excluded from resolution, so a malicious release must survive public scrutiny before it can
enter a lockfile.

The cooldown is configured per ecosystem, sized by how fast that ecosystem detects and
corrects a compromise rather than by blast radius:

| Surface | Window | Mechanism |
| --- | --- | --- |
| npm | 7 days | `min-release-age` in each `.npmrc` (npm 11 native) |
| Go modules | 7 days | `scripts/go-cooldown`, gating the direct requirements a change adds or raises |
| CLI tools resolved by mise | 14 days (GitHub release) / 7 days (package registry) | `scripts/tool-cooldown`, reading `mise.toml` |
| CLI tools installed from PyPI | 7 days | `scripts/tool-cooldown`, reading the `python/*.in` declarations ([ADR-0075](0075-mise-ssot-drift-gate.md)) |
| GitHub Actions | 14 days | `PIN_ACTIONS_MIN_AGE_DAYS`, enforced by `scripts/pin-actions` |
| Container images | 14 days | `PIN_IMAGES_MIN_AGE_DAYS`, enforced by `scripts/pin-images` |
| Dependabot | 5 / 7 / 30 days (patch / minor / major) | cooldown in `.github/dependabot.yml` |

The two tiers in the mise row come from the backend rather than the tool: a version resolved
through a GitHub release gets the same window as `pin-actions` / `pin-images`, because a tag can
be moved onto a different commit, while a version resolved through a package registry gets the
shorter window, because a published version there is immutable.

`capability-diff.yaml` runs `capslock` on dependency-changing PRs as a **supplementary**
signal on the Go side only. It is report-only and is not treated as a detector.

## Consequences

### Positive Consequences

- No third party receives this repository's dependency graph, and CI gains no external
  service dependency.
- The mitigation applies uniformly to every ecosystem, including the npm side where no
  detector exists.
- A cooldown degrades gracefully: it is a delay, not a classifier, so it has no false
  positives to triage and no model to keep current.

### Negative Consequences

- **The cooldown protects the resolution moment, not the installed state.** It closes the
  window in which a fresh malicious version can enter a lockfile; it does nothing about one
  that is already pinned, and nothing about a compromise discovered after the window
  elapses. In the `@asyncapi` case the 7-day npm window had already passed by the time the
  compromise was public — removal upstream is what protected consumers, not this control.
- Security updates are delayed by the same window they impose (Dependabot's security
  updates deliberately bypass their cooldown for this reason).
- **Language runtimes (`go` / `node` / `python`) are outside every window.** A compromised
  runtime distribution is a failure of the language's trust model rather than of one supply-chain
  link, and a delay protects nothing against it; runtimes are governed instead by the separate
  policy of waiting for an LTS. Stated here so the gap reads as a decision rather than an
  oversight.
- `capslock` covers Go only, so the npm half of the graph has no capability signal at all.

## Alternatives Considered

### Commercial malicious-package scanning (Socket / Snyk / Phylum)

Rejected. All require uploading the dependency graph to a vendor and introduce a paid
external dependency into CI. ADR-0001 rules out that class of lock-in for a template whose
adopters cannot be assumed to hold the same vendor contracts.

### Pinning every transitive dependency by hash and reviewing every lockfile change by hand

Rejected as unenforceable at this graph size (the npm tooling tree alone is ~500 packages).
The narrower version of this idea — verifying that lockfile `resolved` URLs point at the
official registry over HTTPS — is adopted instead as `lockfile-integrity.yaml`, which
catches lockfile poisoning without requiring human review of every entry.

### Lengthening the cooldown windows

Rejected as a false sense of safety. The `@asyncapi` compromise was public 12 days after
publication; a longer window would have covered that instance and still missed a slower
one, while delaying every legitimate security patch. The windows are sized to detection
latency, and the residual risk is accepted here explicitly rather than papered over.

## Notes

- Cooldown windows and their rationale: `docs/rules.md`, `.npmrc` in each npm project,
  `.github/dependabot.yml`.
- Related: [ADR-0085](0085-sha-pinned-actions.md) (SHA pinning),
  [ADR-0084](0084-multi-layer-security-scanning.md) (the scanning layers this sits beside),
  [ADR-0001](0001-avoid-lock-in.md) (why SaaS detectors are out).
- The capability signal it is *not* a substitute for: `.github/workflows/capability-diff.yaml`.
