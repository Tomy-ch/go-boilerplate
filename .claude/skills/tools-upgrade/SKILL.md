---
name: tools-upgrade
description: Audit `mise.toml` `[tools]` entries against upstream latest versions, with a configurable supply-chain quarantine. For each tool the latest release is fetched from its backend (GitHub Releases for `aqua:` / `go:` tagged modules, npm registry for `npm:`, PyPI for `pipx:`, language download manifests for `go` / `node` / `python`). Releases newer than `min_age_days` are reported as informational only — never applied automatically — to avoid pulling in newly-published malicious versions before upstream has time to detect and revoke them. Confirms `min_age_days` and the per-tool update set via `AskUserQuestion`, rewrites approved entries in `mise.toml` atomically, runs `make sync-versions` if `go` / `node` / `python` changed, and verifies with `make lint` + `make test`. Use this skill on a routine cadence (monthly / quarterly) or after a security advisory.
---

# Tool Version Upgrade

This skill audits `mise.toml` against upstream latest versions for every tool in the `[tools]` table, with a **supply-chain quarantine gate**: releases newer than `min_age_days` are surfaced as informational only and are never applied automatically. The gate exists because malicious uploads to npm / PyPI / Go module proxies are typically detected and revoked within hours to days; waiting reduces exposure.

A Japanese reference translation is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

Use this skill when:

- Routine periodic (monthly / quarterly) check of pinned tool versions
- Before a release, to confirm there are no known CVEs patched since the current pins
- After a security advisory, to see whether the relevant tool can be updated

Do NOT use this skill for:

- Upgrading Go itself — use `/go-upgrade` (different downstream sync via `make sync-versions`)
- Updating Go module dependencies (`go.mod` `require` block) — use `make tidy-lib` directly
- One-off ad-hoc version bumps — just edit `mise.toml` and run `make sync-versions`

## First Step: Confirm `min_age_days`

This skill **MUST call `AskUserQuestion` immediately after invocation** to confirm the quarantine threshold.

Procedure:

1. If a value is present in the skill arguments (e.g., `/tools-upgrade 14`), include it as a candidate in the question (e.g., "Candidate: `14`").
2. Always invoke `AskUserQuestion`:
    - Question: "Specify the minimum age (in days) for a release to be eligible for auto-apply. Recommended: `7`."
    - Default candidate: `7`
3. Validate the answer is a non-negative integer. Use it as `<MIN_AGE_DAYS>` throughout the rest of the procedure.

Do NOT fetch any upstream API or read `mise.toml` until `<MIN_AGE_DAYS>` is confirmed.

## AI Modification Scope

Per the "Exception: Skill Execution" clause in `CLAUDE.md`, the following paths are permitted to be modified while this skill is running:

- `mise.toml` (the `[tools]` table — write only entries the user explicitly approved)
- `go.mod`, `docker/**/Dockerfile`, `docker/**/README.md`, `docker/**/README.ja.md` — only as the downstream output of `make sync-versions` (the script handles these atomically)
- `docker/**/Dockerfile` `FROM` `@sha256:...` digests + `docker/images-pin.toml` — only via `make pin-images-apply` / `pin-images-resolve`, when a `go` / `node` / `python` runtime bump changed a base-image tag

The following remain protected even during skill execution:

- `AGENTS.md` / `CLAUDE.md`
- Generated files (`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, generated content under `docs/`)
- Any file unrelated to the version bump

## Execution Steps

### 1. Parse `mise.toml`

Read `mise.toml` and enumerate every key under `[tools]`. For each key, determine the backend:

| Key format | Backend | Latest-version source |
| --- | --- | --- |
| `aqua:owner/repo` | aqua (GitHub Releases) | `gh api repos/owner/repo/releases/latest` |
| `go:path/to/module` | go install | GitHub Releases (if hosted there) or `go list -m -versions path/to/module` |
| `npm:package` | npm | `https://registry.npmjs.org/{package}` |
| `pipx:package` | pipx (PyPI) | `https://pypi.org/pypi/{package}/json` |
| Short name (e.g., `golangci-lint`) | mise registry default | Resolve via `mise registry`, then query the resolved backend |
| `go` (runtime) | language download manifest | `https://go.dev/dl/?mode=json` |
| `node` (runtime) | language download manifest | `https://nodejs.org/dist/index.json` |
| `python` (runtime) | language download manifest | `https://www.python.org/api/v2/downloads/release/` |

For each tool, fetch:

- **Latest stable version** (skip pre-release tags: `-rc`, `-beta`, `-alpha`, `-pre`, `-dev`, etc.)
- **Release date** (ISO 8601 timestamp)

Prefer the `gh` CLI (it handles `GITHUB_TOKEN` automatically and raises the rate limit). For non-GitHub endpoints, use `curl -fsSL`.

### 2. Classify

For each tool:

| Class | Condition |
| --- | --- |
| **up-to-date** | `pinned == latest` (after normalizing the optional leading `v` prefix) |
| **eligible** | `pinned != latest` AND `now - release_date >= MIN_AGE_DAYS` |
| **pending** | `pinned != latest` AND `now - release_date < MIN_AGE_DAYS` |
| **resolution_failed** | Backend lookup failed (network error, 404, parsing failure) |

Sanity rule: refuse to "upgrade" to a strictly lower version per semver — if the parsed latest is `<` the pinned version, classify as `resolution_failed` with reason "potential downgrade".

### 3. Display Summary

Print a Japanese-language summary grouped by class. Example:

```text
ツールバージョン監査結果（min_age_days = 7）

✅ 更新候補（公開から 7 日以上経過 / supply-chain quarantine 通過）:
  - golangci-lint: 2.12.2 → 2.13.0 （公開 2026-05-18, 17 日前）
  - sqlc: 1.31.1 → 1.32.0 （公開 2026-04-29, 36 日前）

⚠️ supply-chain quarantine（公開から 7 日未満、通知のみ）:
  - air: 1.65.3 → 1.66.0 （公開 2026-06-02, 2 日前）

✓ 既に最新:
  - oapi-codegen 2.7.0
  - lefthook 2.1.8
  ... (省略可)

❌ 取得失敗:
  - pipx:sqlfluff: PyPI への接続失敗
```

### 4. Confirm Per-tool Update Set

If **eligible** is empty, skip to step 6 with no writes.

Otherwise invoke `AskUserQuestion` with `multiSelect: true`. Each option corresponds to one eligible tool, with the version diff and release date as the description. Default state: all selected.

The user may deselect individual entries (e.g., if a specific bump is known-broken).

### 5. Update `mise.toml`

For each approved tool:

- Locate the exact line in `mise.toml`
- Replace the version literal only — preserve the original key (`aqua:owner/repo` / `go:path/to/module` / short name) and the original `v`-prefix convention if any
- Do not reorder keys, do not touch unrelated keys, do not touch the `[settings]` table

After computing all approved changes, write `mise.toml` **once** (atomic single-pass write). Read the file → apply all replacements in memory → write.

### 6. Run `make sync-versions` if Necessary

If any of `go` / `node` / `python` was updated, run `make sync-versions`. This propagates the new runtime version to `go.mod` and the hardcoded `FROM golang:` / `FROM node:` / `FROM python:` references in the Dockerfile and `docker/**/README.md` files.

If only non-runtime tools were updated, skip `make sync-versions`.

### 6.5. Re-pin Base Image Digests if a Runtime Changed

If step 6 ran `make sync-versions` (i.e. a `go` / `node` / `python` bump changed a `FROM` **tag**), the previously-pinned `@sha256:...` digest now points at the OLD image — a tag/digest mismatch (Docker honors the digest). Re-pin from the registry (this is the `pin-images` skill's job, chained here):

```sh
make pin-images-resolve   # run `docker login` first if Docker Hub returns 429
make pin-images-apply
make pin-images-check
```

The just-published runtime image is normally inside the `PIN_IMAGES_MIN_AGE_DAYS` cooldown, so `pin-images-apply` strips the stale digest and leaves the `FROM` on the tag only (quarantined) — `pin-images-check` accepts that. Report it and note that `/pin-images` should be re-run once the image ages. Skip this step entirely when step 6 was skipped (no runtime change → no tag change → digests still valid).

### 7. Verify

```sh
make lint
make test
```

Report the result table to the user (OK / FAIL per command). Do NOT automatically roll back on failure — the user decides whether to amend, revert, or proceed.

### 8. Final Report

Summarize:

- Number of tools updated
- Number quarantined (pending, not applied)
- Verification result
- Any failures to surface

Do NOT commit, stage, or push. The user reviews the resulting working tree and runs `/commit` (or similar) manually.

## Notes

- **Supply-chain quarantine rationale**: typical "dependency confusion" / "malicious release" attacks (e.g., npm `ua-parser-js` 2021, PyPI `ctx` 2022) were detected and yanked within 24–72 hours. A 7-day quarantine catches the vast majority while still being responsive for routine bumps.
- **Pre-release exclusion**: this skill always selects the latest **stable** release. Pre-release tags are visible in upstream but never chosen as `latest`.
- **Calendar versioning**: for tools using calendar versioning (e.g., `2024.12.30`), comparison is lexicographic with semver fallback. The "potential downgrade" guard remains active.
- **Rate limits**: GitHub API anonymous limit is 60 req/h per IP. The skill SHOULD use `gh api` which authenticates via `GITHUB_TOKEN` (1000 req/h authenticated).
- **Idempotency**: multiple invocations are safe. A second run after a successful apply will show those tools as up-to-date.
- The skill never auto-pushes. The user reviews the working tree, runs `make sync-versions` if needed, then commits and pushes manually.

## Checklist

Confirm the following before reporting completion:

- [ ] `<MIN_AGE_DAYS>` confirmed via `AskUserQuestion`
- [ ] Every `[tools]` entry's backend was resolved (or surfaced as resolution_failed with a reason)
- [ ] Each tool was classified (up-to-date / eligible / pending / resolution_failed)
- [ ] Classification table presented to the user
- [ ] If eligible set non-empty: user confirmed per-tool update set via `AskUserQuestion`
- [ ] `mise.toml` rewritten atomically with only approved changes, preserving key formats and `v`-prefix convention
- [ ] `make sync-versions` run if go / node / python was updated
- [ ] If a runtime was bumped: base image digests re-pinned (`make pin-images-resolve` + `pin-images-apply` + `pin-images-check`); quarantined-until-aged images reported for a later `/pin-images` re-run
- [ ] `make lint` + `make test` run after writes
- [ ] Final result table reported to the user
- [ ] After updating `SKILL.md`, also update `SKILL.ja.md` to keep the Japanese translation in sync
- [ ] No commit / stage / push performed
