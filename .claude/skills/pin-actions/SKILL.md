---
name: pin-actions
description: Pin GitHub Actions `uses:` references to immutable commit SHAs to harden against tag-hijacking supply-chain attacks. Backed by the `scripts/pin-actions` Go tool (run via `make pin-actions-resolve` / `make pin-actions-apply`). The `resolve` phase scans `.github/workflows/**` and `.github/actions/**` for external action references, resolves each `owner/repo@<tag>` to its commit SHA with `git ls-remote` (annotated tags are dereferenced via `^{}`), and writes them to a `.github/actions-pin.toml` lockfile that is the single source of truth. The `apply` phase rewrites each `uses: owner/repo[/sub]@<ref>` to `uses: owner/repo[/sub]@<sha> # <ref>` from the lockfile. Idempotent — already-pinned refs (`@<sha> # <tag>`) are re-resolved from the `# <tag>` comment, so re-running refreshes SHAs (complements Dependabot). Local `./` action refs have no `@ref` and are skipped; the codeql-action subpaths (init / analyze / …) collapse to one `github/codeql-action@<tag>` lockfile entry. The skill orchestrates resolve → lockfile review → apply → actionlint verification → commit. Use to first-pin all actions, when adding/updating an action, or on a routine hardening cadence.
---

# Pin GitHub Actions to commit SHAs

Mutable tag references (`actions/checkout@v6`, `jdx/mise-action@v2`) let an attacker who can move a tag run arbitrary code in CI. Pinning each action to an immutable commit SHA (`@<sha> # v6`) removes that risk while keeping the human-readable tag in a trailing comment.

A Japanese reference translation can be generated with the `canonicalize-doc` skill (not yet present).

## When to Use

- First-time hardening: pin every action across the repo.
- After adding a new action or bumping an action's tag.
- On a routine cadence to refresh pinned SHAs (alongside Dependabot, which updates SHA-pinned `uses:` while preserving the `# <tag>` comment).

## How it works

Two phases, both deterministic and backed by `scripts/pin-actions` (host `go run`, like `sync-versions`):

| Phase | Command | Effect |
| --- | --- | --- |
| resolve | `make pin-actions-resolve` | Scan `uses:` → resolve tag→SHA via `git ls-remote` → write `.github/actions-pin.toml` (SSOT) |
| apply | `make pin-actions-apply` | Rewrite `uses: …@<ref>` → `uses: …@<sha> # <ref>` from the lockfile |

`git ls-remote` reads public repos with no auth. Whitespace handling is line-internal only, so blank lines / step structure are preserved (the apply diff is `uses:`-line-only).

## Procedure

1. **Resolve.** Run `make pin-actions-resolve`. It (re)writes `.github/actions-pin.toml` with one `"owner/repo@tag" = "<sha>"` entry per distinct action ref. New actions are added; already-pinned refs are re-resolved from their `# <tag>` comment (so this also *updates* SHAs).

2. **Review the lockfile.** Inspect `.github/actions-pin.toml`. Sanity-check that every expected action is present, the tags look right, and each SHA is a 40-hex commit. This is the audit point — the lockfile is the single source of truth for what `apply` will write.

3. **Apply.** Run `make pin-actions-apply`. It rewrites every `uses:` ref from the lockfile to `@<sha> # <tag>`. If any ref is missing from the lockfile it aborts and asks you to re-run resolve.

4. **Verify.** Confirm the change is safe:
   - `git diff --stat .github/workflows .github/actions` — insertions should equal deletions (pure line replacements; no structural churn).
   - Run actionlint (`make actions-lint`) — the pinned workflows must still parse and validate.

5. **Commit.** Commit the lockfile and the pinned workflow/action files together (e.g. `CI: GitHub Actions を commit SHA に固定`). Per `CLAUDE.md`, split scope appropriately and use the Co-Authored-By footer.

## Notes

- **Idempotent.** Re-running resolve → apply with no upstream tag movement produces no diff.
- **Updating pins.** To pull newer SHAs for the same tags, just re-run resolve (it re-resolves from the `# <tag>` comment) then apply.
- **Scope.** Only external `owner/repo[/sub]@<ref>` refs are touched. Local composite actions (`uses: ./.github/actions/...`) have no `@ref` and are skipped.
- **Subpaths.** `github/codeql-action/init@v4`, `.../analyze@v4`, etc. share a single `github/codeql-action@v4` lockfile entry (same repo, same SHA).
- Enforcing that actions *stay* pinned (a `pinact run --check`-style CI gate) is intentionally out of scope here; this skill applies pinning, it does not block PRs.
