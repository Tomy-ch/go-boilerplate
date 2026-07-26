---
name: dep-vuln-upgrade
description: Patch only the npm or Go dependencies named in a CVE, GHSA, npm audit, Dependabot, Trivy, or govulncheck advisory. Use to make targeted vulnerability fixes, including transitive npm dependencies that need scoped overrides and indirect Go modules. Select the minimal safe fixed version, respect repository release-age controls, regenerate lockfiles through package managers, and verify the resolved advisory state. Do not use for blanket dependency or Go-version upgrades.
---

# Targeted Dependency Vulnerability Upgrade

Accept an advisory list. Parse package, installed version, fixed candidates, advisory ID, and severity. Deduplicate packages with several advisories. Touch only named packages and their mechanically required lockfiles, vendor output, or generator output.

## Plan before writes

1. Locate each package in every `package-lock.json` or in `go.mod`. Classify it as direct or transitive/indirect. Report absent packages; never guess a location.
2. Choose the smallest fixed version on the installed major line. Do not downgrade. Flag a required major upgrade.
3. Read the lockfile directory's `.npmrc`. Its `min-release-age=N` is a hard npm resolution constraint. For Go or npm without a cooldown, use a seven-day caution window unless the user supplied another value.
4. Fetch the chosen version's publication date. Classify it:

   - `clear`: old enough and non-major; apply.
   - `too-new`: eligible but requires user opt-in.
   - `blocked`: inside an npm `min-release-age` window; defer it and report when it becomes eligible.
   - `major`: requires explicit user approval even when old enough.

Present the plan in Japanese. Apply only clear, same-major fixes without a further question. Ask for approval only for `major` or `too-new` entries.

## Apply approved fixes

- For direct npm dependencies, update the declared version and run `npm install --package-lock-only` in that package directory.
- For transitive npm dependencies, add the narrowest parent-scoped override using a same-major floor (`>=<fixed> <<next-major>`), preserving existing overrides, then run `npm install --package-lock-only`. Use an exact override only when a documented newer version is known broken. Never edit `package-lock.json` by hand.
- For Go modules, run one batched `go get module@version ...`, then `go mod tidy`; run `go mod vendor` only when `vendor/modules.txt` exists. Never hand-edit vendor output.
- Re-read the installed lockfile version after npm resolution. If a range selected a newer, too-new version, pin the approved exact direct version and regenerate.

## Verify

Run `npm audit` for each affected npm package and `govulncheck ./...` plus `go build ./...` for Go changes. Run package typechecks/tests for major npm upgrades. Regenerate only the artifacts driven by a changed generator dependency, using its existing make target, then check for drift.

Report applied packages, advisory IDs, temporary overrides to reclaim after the parent is fixed, deferred items, and every verification result in Japanese. Do not stage, commit, push, lower a release-age control, or upgrade unrelated packages.
