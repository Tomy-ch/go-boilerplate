---
name: actions-pin
description: Audit and update SHA-pinned GitHub Actions in `.github/workflows/**` and `.github/actions/**` using this repository's quarantine-aware lockfile workflow. Use for routine Actions pin refreshes, a GitHub Actions security advisory, or a requested major upgrade. Default to same-major updates; accept `major` for major upgrades and `days=N` (default 14) for the minimum release age.
---

# GitHub Actions Pin Update

Use the trailing tag comment in each external `uses:` reference as the intended version and `.github/actions-pin.toml` as the resolved SHA lockfile.

```yaml
uses: owner/repo@<40-hex-sha> # <tag>
```

`make pin-actions-resolve` resolves eligible tags into the lockfile, and `make pin-actions-apply` updates only the SHAs. Never update an Action release newer than the exclusion window unless the user explicitly passes `days=0`.

## Inputs

Parse invocation tokens in any order.

- No argument: update within existing major versions; minimum age is 14 days.
- `major` or `--major`: consider newer major versions too.
- `<integer>`, `days=N`, or `--days N`: set the minimum release age. Reject negative values. Warn explicitly for `0` because it disables quarantine.

Do not use this skill for `mise.toml`, Go, or Go module dependency upgrades. Do not change local composite actions (`uses: ./...`).

## Workflow

1. Read `AGENTS.md`, `.github/actions-pin.toml`, and all external `uses:` lines in `.github/workflows/` and `.github/actions/`. Identify each Action, all locations, current tag comment, and current major.
2. Ensure the repository can run the pin tooling. It uses `go run ./scripts/pin-actions` and requires a vendor tree consistent with `go.mod`. If that is the only blocker, run `go mod vendor`. Obtain an authenticated GitHub token if available (`gh auth token`) so release-date queries do not hit anonymous rate limits.
3. For each distinct Action, query non-prerelease releases and their publication dates. Also verify that a moving major tag exists before selecting it.
4. Select a target for major `M`, where `M` is the current major unless `major` was requested:
   1. Prefer moving tag `vM` when its resolved head is older than the cutoff.
   2. Otherwise choose the newest exact `vM.x.y` release older than the cutoff.
   3. If no aged release exists in that major, hold the current pin and record why.
   Use the upstream tag spelling exactly; some projects change between `0.x.y` and `v0.x.y`.
5. For every proposed major upgrade, inspect the upstream release notes or `action.yml` and compare all repository `with:` inputs. Hold the Action if any input is incompatible; do not silently alter workflow behavior to accommodate it.
6. Present the concrete plan in Japanese before writing: changes (including moving versus exact pin and release age), held items and reasons, and unchanged items. Ask the user to confirm the proposed set. Do not write until confirmation.
7. For confirmed items only, edit the trailing `# <tag>` comment. Match the full `uses:` line when multiple Actions share a tag. Do not edit held or unchanged entries.
8. Resolve and apply:

   ```sh
   export GITHUB_TOKEN="$(gh auth token)"
   make pin-actions-resolve PIN_ACTIONS_MIN_AGE_DAYS=<days>
   make pin-actions-apply
   ```

   If the moving tag is still fresh, retaining an existing within-major pin is expected. If resolution reports a missing moving tag, use an eligible exact release instead.
9. Verify:

   ```sh
   make pin-actions-check
   make actions-lint
   ```

   Report each result. Do not automatically roll back on failure.

## Safety and Completion

- Modify only `.github/workflows/*.{yml,yaml}`, `.github/actions/*/action.{yml,yaml}`, and `.github/actions-pin.toml` while this skill runs.
- Do not change `with:` inputs, workflow step logic, `scripts/pin-actions`, generated files, or `AGENTS.md`.
- Do not stage, commit, or push. Report exact-version step-back pins so they can be revisited after the moving tag ages.
- Treat `make pin-actions-check` and `make actions-lint` as required completion checks.
