---
status: accepted
date: 2026-08-11
deciders: [maintainers]
tags: [dependencies, security, ci, setup-review]
---

# ADR-0107: Resolve every Node package with pnpm; do not use npm

## Status

accepted

## Context

This repository carries two Node packages — `scripts/` (repository tooling) and `docs-viewer/`
(the documentation portal frontend). They are
deliberately separate packages with separate lockfiles: a dependency change is reviewed next to
the code that uses it, and the viewer's dependency graph does not reach the tooling that gates CI.

Nothing forces them to share a package manager, and for a period they did not. The question this
record answers is whether "whichever resolver each package happened to start with" is an
acceptable steady state.

It is not, and the reason is narrower than a preference for one CLI over another. The supply-chain
control this repository relies on is a **publication cooldown** ([ADR-0090
(malicious-package-detection-via-cooldown)](0090-malicious-package-detection-via-cooldown.md)):
a version published inside a window cannot enter a lockfile, because the window buys the time in
which a malicious publish is typically detected and revoked. The two resolvers enforce that window
in materially different places.

npm applies `min-release-age` **while resolving**. `npm ci` does not resolve — it replays the
lockfile — and every CI job and image build replays. So an in-window version that once reached the
lockfile is invisible from then on: it installs cleanly, reports nothing, and after the window
passes it is indistinguishable from a normally resolved entry. A deliberate override leaves no
trace anywhere.

pnpm re-verifies the **entire lockfile** against the active policy on every install, including the
`--frozen-lockfile` replay path. Taking an in-window version requires a `minimumReleaseAgeExclude`
entry naming the exact version, and without one every later install fails — CI included. The entry
lives in a tracked, `CODEOWNERS`-covered file, so the override is reviewed by construction.

The same asymmetry repeats for lifecycle scripts. pnpm's `strictDepBuilds` + `allowBuilds` fail the
install when a dependency that was never reviewed brings a build script; npm's only lever is
`--ignore-scripts`, which skips scripts silently rather than surfacing the new one.

Keeping npm therefore cost a compensating control that existed for no other reason: an audit that
read each lockfile against its own `.npmrc`, reported entries younger than the window, and
deliberately never failed the build. It could only report, because by the time it looked the
decision had already been made and replayed.

## Decision

**Every Node package in this repository resolves with pnpm.** Each carries its own
`pnpm-workspace.yaml` (policy) and `pnpm-lock.yaml` (resolution).

npm is not used as a resolver anywhere — not in a package, not in a workflow, not in an image
build. `package-lock.json` and `.npmrc` are not present, and adding one reintroduces a lockfile
whose cooldown cannot be re-checked after the fact.

A new Node package copies the policy block from an existing `pnpm-workspace.yaml` rather than
declaring a reduced one; the settings are the control, not boilerplate.

Two consequences that look like conventions but follow from the decision:

- **`--ignore-scripts` is not added to a `pnpm install`.** It suppresses the `strictDepBuilds`
  failure, which is the signal that an unreviewed build script appeared. Static analysis asks for
  the flag on sight (`docker:S6505` / `githubactions:S6505`); those rules are excluded by rule key
  in `sonar-project.properties` rather than followed.
- **`npm audit` output remains a valid input.** Reading a report a person pasted is not the same as
  running npm as a resolver.

## Consequences

### Positive

- The cooldown holds on the replay path, so an in-window version cannot enter through a lockfile
  that CI never re-examines.
- Taking an in-window version becomes a reviewed edit to a tracked file instead of an act detected
  afterwards, if at all.
- The bespoke npm cooldown audit — a tool, a make target, a workflow, and its egress declaration —
  is deleted rather than maintained. There is nothing to detect after the fact.
- One resolver means one set of policy keys to understand, and CI stops carrying a per-package
  branch in every job that installs.

### Negative

- pnpm's isolated `node_modules` refuses a package that a flat npm layout would have resolved by
  accident. That is a correctness gain, but it surfaces as breakage when a dependency's own
  declaration is incomplete.
- A runtime image cannot simply run `npm ci`; it needs pnpm, which arrives through the pinned
  toolchain and costs a build stage that exists only to resolve dependencies.
- The repository is now exposed to pnpm as a single upstream. A regression in its resolution policy
  affects all three packages at once, where two resolvers would have contained it.

### Neutral

- The window itself is unchanged at 7 days; only where it is enforced changed. pnpm states it in
  minutes, so the declaration reads `minimumReleaseAge: 10080`.
