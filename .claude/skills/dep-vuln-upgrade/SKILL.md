---
name: dep-vuln-upgrade
description: Patch specific vulnerable dependencies flagged by a security advisory (CVE / GHSA) across this repo's npm lockfile (`mock-auth-server/package-lock.json`) and the Go module graph (`go.mod` / `go.sum`), targeting only the named packages rather than a blanket upgrade. Use this skill whenever the user pastes a vulnerability report — `npm audit`, Trivy, Dependabot, `govulncheck`, or a hand-written list of "package current → fixed (CVE)" lines — and wants those exact packages bumped to a fixed version, including transitive npm deps that need an `overrides` pin and indirect Go modules. It locates each package's ecosystem and lockfile, picks the minimal same-major fixed version, aligns its supply-chain age gate to the repo's own `.npmrc` `min-release-age` cooldown (deferring any fix that npm would hard-reject as too new), chains `/supply-chain-triage` on every entry the cooldown catches so the opt-in rests on a scored evidence verdict rather than on a day count alone, then auto-applies the safe `clear` patches while asking (via `AskUserQuestion`) only about major-version bumps and too-new opt-ins. It updates lockfiles with `npm install --package-lock-only` (scoped `overrides` for transitive deps), runs `go get` + `go mod tidy` + `go mod vendor` for Go modules, and verifies with `npm audit` / `govulncheck` / build plus generated-artifact drift checks. Do NOT use it for routine "bump every tool to latest" audits (that is `/tools-upgrade` for `mise.toml`), for upgrading the Go language version (`/go-upgrade`), for a general `go.mod` dependency refresh (`make tidy-lib`), or for the pnpm-resolved packages (`scripts/` and `docs-viewer/`), whose `overrides` live in `pnpm-workspace.yaml` and are not covered by this flow.
argument-hint: '[advisory list — one "package current → fixed (CVE/GHSA)" per line] [min_age_days]'
---

# Dependency Vulnerability Upgrade

This skill takes a **security advisory list** and patches only the named vulnerable dependencies to a fixed version, spanning two ecosystems this repo uses:

- **npm** — dependencies recorded in each `package-lock.json` (currently `mock-auth-server/` only), including **transitive** deps that must be pinned via a `package.json` `overrides` entry.
- **Go** — modules in `go.mod` / `go.sum`, including **indirect** dependencies.

Two packages resolve with **pnpm** instead — `scripts/` and `docs-viewer/`, each carrying its own
`pnpm-lock.yaml` and `pnpm-workspace.yaml`. The step-by-step flow below is written for npm and does
**not** cover them: pnpm keeps its `overrides` in `pnpm-workspace.yaml` rather than in
`package.json`, and its `parent>child` selector constrains only the direct edge where npm's nested
override applies to the whole subtree, so an entry cannot be copied across unchanged. Treat a pnpm
package as out of this skill's automation until the flow is extended — report the finding, and let
the maintainer edit `pnpm-workspace.yaml` deliberately.

It is deliberately **targeted**: it changes only the packages named in the advisory, never a blanket "everything to latest". That keeps a security patch reviewable and decoupled from unrelated churn. For blanket upgrades use `/tools-upgrade` (mise tools) or `make tidy-lib` (Go modules) instead.

A Japanese reference translation is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

Use this skill when:

- The user pastes a vulnerability report (`npm audit`, Trivy, Dependabot alert, `govulncheck`, or a hand-written list) and wants the flagged packages patched.
- A CVE / GHSA advisory names a package that lives in a `package-lock.json` or in `go.mod`, and you want the minimal fixed-version bump plus verification.
- Transitive npm deps (pulled by a tool like `redocly` / `orval`) need forcing to a patched version via `overrides`.

Do NOT use this skill for:

- Routine "bump all pinned tools to latest" — that is `/tools-upgrade` for `mise.toml` `[tools]`.
- Upgrading the Go language version — that is `/go-upgrade` (different downstream sync).
- A general `go.mod` dependency refresh unrelated to a specific advisory — use `make tidy-lib`.
- npm packages that are actually mise-managed tools — those belong to `/tools-upgrade`.

## First Step: Parse Advisories and Resolve the Caution Threshold

Parse the advisory list, then resolve the supply-chain caution threshold `<MIN_AGE_DAYS>` — **preferring the repo's own npm cooldown so the skill and the toolchain agree**, and asking the user only when nothing authoritative is on disk.

Procedure:

1. Parse the advisory list from the skill arguments or the most recent user message. Each entry yields: **package name**, **current version** (if given), **candidate fixed version(s)** (there may be several, one per major line), **CVE/GHSA id**, and **severity**. The list may be free-form — tolerate the common shapes (`- [HIGH] lodash 4.17.23 → 4.18.0 (CVE-...)`, `npm audit` blocks, Trivy rows). If an entry's ecosystem or location is ambiguous, resolve it in Step 1 (do not guess here).
2. Detect the repo's npm cooldown: read every `.npmrc` under the lockfile dirs (e.g. `mock-auth-server/.npmrc`) for a `min-release-age=N` line. This is npm 11+'s native supply-chain quarantine — it applies a hard `before = now − N days` cutoff at **dependency resolution** time (`npm install` / `npm install --package-lock-only`), so a version newer than the cutoff **cannot be installed at all**, not merely flagged. If found, adopt `N` as `<MIN_AGE_DAYS>` for that lockfile so the skill's caution threshold matches the wall the toolchain will actually enforce.
3. If no `.npmrc` cooldown governs a given change (e.g. a Go module, or a lockfile with no `min-release-age`), use `7` as the default `<MIN_AGE_DAYS>` for it, unless a value was passed in arguments. Only call `AskUserQuestion` to confirm the threshold when there is genuine ambiguity (conflicting `.npmrc` values, or the user asked to override); a lone repo `.npmrc` value or the `7` default does not need a question — proceed and state the value you used.

The threshold plays two different roles depending on ecosystem:

- **npm under a `.npmrc` `min-release-age`**: a **hard block**. A fixed version inside the cooldown will make `npm install` fail with `ETARGET ... No matching version found ... with a date before <cutoff>`. Do not fight the repo's own policy — treat such a version as **deferred** (Step 4), not applied.
- **everywhere else (Go, npm with no cooldown)**: a **caution flag, not a hard block** — because the point is to fix a known vulnerability, a too-new fixed version is surfaced and confirmed, not silently withheld.

Do NOT fetch registries or edit any file until `<MIN_AGE_DAYS>` is resolved.

## AI Modification Scope

`AGENTS.md` normally treats `docker/**` and repository-root files as out of scope for AI edits. **Invoking this skill is the explicit user instruction that relaxes that**, but only for the specific files a dependency patch must touch, and only for this run. This is a documented, non-loophole exception per the "Skills must not be a loophole" clause in `AGENTS.md` — the paths below are the entire surface, and the user is told which ones changed.

Permitted to modify while this skill runs:

- `**/package.json` — only to add/adjust a `dependencies` version or an `overrides` entry for an approved package (the npm one here is `mock-auth-server/package.json`; `scripts/package.json` is pnpm and its `overrides` live in `scripts/pnpm-workspace.yaml`, which this skill does not edit).
- `**/package-lock.json` — the deterministic output of `npm install` in that package directory for approved changes.
- `go.mod` / `go.sum` — the output of `go get <module>@<version>` + `go mod tidy` for approved Go modules.
- `vendor/**` — **only** as the mechanical output of `go mod vendor`, and **only if the repo vendors** (a `vendor/modules.txt` exists). A `go.mod` bump leaves `vendor/modules.txt` out of sync; the build then fails with `inconsistent vendoring`, so re-vendoring is a required downstream step, not an edit of taste. Never hand-edit vendored files.
- Regenerated artifacts that these dependencies drive, **only** when a drift check (Step 6) shows they moved — e.g. `mock-auth-server/openapi/openapi.gen.yaml` / `src/generated/**` via `make gen-mock-auth-oapi`. Regenerate with the repo's `make` target; never hand-edit a generated file.

Hard-protected even during this skill (never touch):

- `AGENTS.md` / `CLAUDE.md`
- `node_modules/**` (build product, not tracked source), and `vendor/**` **except** via `go mod vendor` as above
- Generated files edited by hand (`**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, generated content under `docs/`) — regeneration via `make` is fine; hand-editing is not.
- Any package NOT named in the advisory list. This skill is targeted; it must not opportunistically bump neighbors.

## Execution Steps

### 1. Locate Each Package

For every advisory entry, find where the package actually lives and classify it. Do not assume — verify against the tree.

```sh
# npm: which lockfile(s) contain the package
find . -name package-lock.json -not -path '*/node_modules/*'
# then, per candidate lockfile:
grep -n "\"node_modules/<pkg>\"" <lockfile>          # presence + installed version
# direct vs transitive: is it in that package.json's own dependencies/devDependencies?
grep -n "\"<pkg>\"" <dir>/package.json

# Go: is the module in go.mod, and is it direct or indirect?
grep -n '<module-path>' go.mod
```

Record per entry:

| Field | How |
| --- | --- |
| ecosystem | `npm` or `go` |
| location | the `package-lock.json` dir, or `go.mod` |
| installed version | from the lockfile / `go.mod` |
| direct / transitive | in the package.json `dependencies`/`devDependencies` (npm) or without `// indirect` (go) → direct; else transitive/indirect |

If a package cannot be found in any lockfile or `go.mod`, report it as **not-present** (already removed, or in a lockfile this repo does not have) and skip it — do not invent a location.

### 2. Select the Fixed Version

For each present package, choose the target version:

- **Default: the minimal fixed version that stays on the currently-installed major line.** From a multi-candidate advisory (`brace-expansion 1.1.15 → 5.0.7 / 1.1.16 / 2.1.2`), pick the one matching the installed major (`1.1.15` → `1.1.16`). This minimizes breaking risk.
- **Major bump required** (the only fix is a higher major, or the installed line has no patched release — e.g. `@hono/node-server 1.19.14 → 2.0.5`): flag it explicitly as **potentially breaking**. It will need per-package confirmation (Step 4) and closer verification (Step 6).
- **Downgrade guard**: never select a version strictly lower than installed. If the only "fix" parses lower, mark it `needs-manual` and surface it rather than applying.
- **Gate the version that actually resolves, not just the advisory's number.** A `^`/`~` range in `package.json` floats to the newest matching patch, which may be far newer than the advisory's fixed version and thus `too-new`. Compute the date on the version the lockfile will land on (Step 5 re-checks this), and when a range would resolve to a too-new / unvetted version in a dir with **no `.npmrc` cooldown** to hold it back, **pin the exact approved fixed version** instead of leaving the caret to float.

Fetch each chosen version's **publish date** to feed the caution gate (Step 3):

```sh
# npm
curl -fsSL https://registry.npmjs.org/<pkg> | jq -r '.time["<version>"]'
# Go (module publish time)
curl -fsSL "https://proxy.golang.org/<module>/@v/<version>.info" | jq -r '.Time'
```

### 3. Apply the Supply-chain Caution Gate

For each chosen version, compute `now - publish_date` and set a disposition:

| Flag | Condition | Effect |
| --- | --- | --- |
| **clear** | `>= MIN_AGE_DAYS` | eligible — apply by default (see Step 4) |
| **too-new** | `< MIN_AGE_DAYS`, npm with **no** `.npmrc` cooldown, or Go | eligible but flagged; ⚠️ surfaced and NOT applied by default — the user must opt in |
| **blocked** | `< MIN_AGE_DAYS` under a `.npmrc` `min-release-age` | **cannot be installed** — the repo's own npm cooldown hard-rejects it. Do not apply; mark **deferred** and report when it will clear (`publish_date + N days`) |

The caution exists because malicious uploads to npm / the Go proxy are typically detected and revoked within hours to days. A security fix is urgent, so `too-new` is a warning the user can override — but `blocked` is the repo's own policy enforced by npm itself, and the skill respects it rather than disabling the cooldown. Note the boundary is real: a version published even a few minutes inside an N-day window is `blocked` until the window rolls past its exact publish timestamp.

### 3.5. Triage What the Gate Caught

For every entry dispositioned `too-new` or `blocked`, chain **`/supply-chain-triage`** (one invocation per entry) before Step 4 presents the decision.

The window is a proxy for four questions — did the publisher change, does the artifact match its source, what actually changed, did new dependencies appear — and those can be answered directly instead of waited out (`docs/design/security.md` → "Dependencies"). This step is where that happens: a `too-new` opt-in is precisely the decision that should rest on evidence rather than on the user's tolerance for a day count, and a `blocked` entry needs to know whether waiting is merely inconvenient or actually protective.

Pass the ecosystem, package, candidate version, the **baseline the lockfile currently holds** (the diff's other end), `<MIN_AGE_DAYS>`, the disposition, and the CVE forcing the move. Triage is report-only — it reads the tarball / module zip without executing it and returns a 0–12 score, a band, and citations. It changes nothing and never adopts anything; the decision stays in Step 4.

Skip it when there is nothing flagged (all `clear`), and skip it for an entry the user has already declined this run. Carry each entry's band into Step 4's summary and option descriptions so the choice is made with the evidence attached.

### 4. Display Summary; Auto-apply Clear, Confirm Only the Flagged

Print a Japanese summary grouped by disposition. Example:

```text
依存脆弱性パッチ監査（min_age_days = 7, mock-auth-server/.npmrc 由来）

✅ 適用（同一 major の最小修正版 / caution 通過 → 確認なしで適用）:
  - lodash 4.17.23 → 4.18.0  [docker/tools, 推移的]  (CVE-2026-4800, CVE-2026-2950 / HIGH)
  - js-yaml 4.2.0 → 4.3.0     [docker/tools, 直接]    (CVE-2026-59869, CVE-2026-53550 / HIGH)

⚠️ major 跨ぎ（breaking の可能性 / 別途確認）:
  - @hono/node-server 1.19.14 → 2.0.5  [mock-auth-server, 直接]  (GHSA-frvp-7c67-39w9 / MEDIUM)

⚠️ too-new（公開が min_age 未満 / 別途 opt-in）:
  - fast-uri 3.1.3 → 3.1.4  [docker/tools]  (公開 3 日 / CVE-2026-16221 / HIGH)
      トリアージ: 1/12 LOW（発行者同一・provenance 一致・差分は URL parser のみ・新規依存なし）

⛔ deferred（repo の .npmrc min-release-age に阻まれ install 不可）:
  - brace-expansion 1.1.16  [docker/tools, spectral-core 内]  (公開が cooldown 内 / 2026-07-22 頃に解除)
      トリアージ: 2/12 LOW（ただし .npmrc の cooldown により install 自体が不可）

❓ 未検出 / 要手動:
  - （lockfile に見つからない等）
```

The confirmation policy is deliberately asymmetric — a clear patch is the whole point of running the skill, so do not make the user click through it:

- **clear AND not a major bump → apply without asking.** These are default patches; confirming each one is friction the user has already implicitly authorized by invoking the skill on the advisory.
- **major-bump (a higher major than installed) → always report separately and confirm**, even when the version itself is `clear`. A major bump can break the code that imports it, so the user decides knowingly. Verify it more closely in Step 6.
- **too-new (caution, no repo cooldown) → report and confirm (opt-in)**, default not applied.
- **blocked / deferred → never applied**; just report it and when it will clear.

Only call `AskUserQuestion` when there is a **major-bump** and/or **too-new** entry to decide (a single `multiSelect: true` question listing just those, all deselected by default). Give each `too-new` option its Step 3.5 triage band in the description (`1/12 LOW` / `7/12 HIGH` / `INSUFFICIENT-EVIDENCE`) — that band is the reason to opt in or wait, so it belongs where the click happens rather than only in the summary above. If every eligible entry is `clear`-and-non-major, apply them straight away with no question. If nothing is eligible, skip to Step 7 with no writes.

### 5. Apply the Updates

Group approved packages by location and apply per group. Never edit a `package-lock.json` by hand — let `npm` regenerate it.

Only `package.json` + `package-lock.json` are tracked here (`node_modules/` is git-ignored and rebuilt in the toolbox image), so update the **lockfile only** with `--package-lock-only` — it resolves from the registry and rewrites the lockfile without downloading the full tree. Do the edit to `package.json` yourself, then let `npm` regenerate the lockfile; never hand-edit `package-lock.json`.

**npm — direct dependency** (in that package.json's `dependencies`/`devDependencies`): bump the declared range, then regenerate.

```sh
# edit <dir>/package.json: "<pkg>": "<new-range>"
cd <dir>
npm install --package-lock-only
```

After regenerating, **read back the version the lockfile actually pinned** (`node -e '...node_modules/<pkg>...'`). A `^`/`~` range resolves to the newest matching patch, so `^2.0.5` can land on a `2.0.11` published yesterday — re-gate that resolved version against Step 3. If it is `too-new` (and the dir has no `.npmrc` cooldown to hold it back), pin the exact approved version (`"<pkg>": "2.0.5"`) and regenerate, so the lockfile lands on the vetted version rather than the freshest one.

**npm — transitive dependency** (pulled by another package; not a direct dep): force it to a patched version with an `overrides` entry. Two decisions matter — the **specifier form** and the **scope**.

**Specifier — prefer a same-major range floor, not an exact pin.** An `overrides` entry is authoritative and *sticky*: npm forces exactly what you write and it never floats up on its own. An exact pin (`"<pkg>": "1.2.3"`) therefore **freezes** that nested copy — when the parent later ships a version that natively carries the fix, the override holds the old branch; and if `1.2.3` itself later gets a CVE, the override keeps forcing the now-vulnerable version (the pin becomes the vulnerability, and an easily-missed one). Write a **floor that stays within the installed major** so the fix is enforced as a minimum while the dep can still track the parent upward:

```jsonc
// <dir>/package.json — scoped floor: at least the patched version, still floats within the major
"overrides": {
  "<vulnerable-parent>": { "<pkg>": ">=<fixed> <<next-major>" }   // e.g. ">=1.2.3 <2"
}
// exact pin ("<pkg>": "<fixed>") ONLY when a newer in-range version is known-broken and must be held
// bare, unscoped "<pkg>": "..." ONLY when every copy must move
```

**Scope — pin under the offending parent**, not globally, so you fix the vulnerable nested copy **without downgrading an already-patched top-level copy** of the same package. A bare global override forces every copy to your specifier.

Then regenerate: `cd <dir> && npm install --package-lock-only`. Add to an existing `overrides` block; do not clobber siblings. Batch all approved changes in one package into a single edit + one `npm install --package-lock-only`. A range floor resolves to the newest in-range version npm will accept, so under a `.npmrc` `min-release-age` it lands on the newest *aged* version at or above the fix — no manual re-pin as the dep moves. If npm rejects even the floor with `ETARGET ... date before <cutoff>`, the fixed version itself is still inside the cooldown (Step 3 `blocked`) — remove that entry, leave the package deferred, and proceed with the rest.

**Go module** (direct or indirect):

```sh
go get <module>@<version>
go mod tidy
go mod vendor        # ONLY if the repo vendors (vendor/modules.txt exists)
```

Batch all approved Go modules into one `go get` (multiple `module@version` args) + a single `go mod tidy`, so `go.sum` settles once. If `vendor/modules.txt` exists, run `go mod vendor` afterwards — without it the build fails with `inconsistent vendoring` because `go.mod` and `vendor/modules.txt` disagree.

### 6. Verify

Run the checks that match what actually changed — a dependency patch rarely touches first-party source, so scope the verification to the ecosystems that moved rather than always running the full suite. Report each as OK / FAIL; do NOT auto-roll-back on failure — the user decides.

Go changes at minimum build and vuln-scan clean (`go build ./...` + `govulncheck ./...`); run `make lint` / `make test` when a Go change is broad enough to warrant the full suite. A **major npm bump** must be verified more closely — run that package's own `typecheck` + tests (e.g. `npm run typecheck` + `npm test` in `mock-auth-server/`), since a major can change the API the code calls.

npm changes — confirm the advisory is actually resolved and the lockfile is clean:

```sh
cd <dir> && npm audit            # the patched CVEs should no longer appear
```

`npm audit` is the source of truth for the **real fix floor**, and it can differ from the user's list. Two cases to surface rather than silently resolve:

- **A higher fix floor than the advisory named.** The package may carry a *second* advisory whose first-fixed version is higher than the one the user pasted (e.g. the list says "fixed in 2.0.5", but `npm audit` still flags a separate moderate advisory affecting `2.0.0 - 2.0.9`, fixed only in `2.0.10`). Do not silently jump to that higher version if it is `too-new` — surface the conflict and whether the vulnerable path is even reachable (e.g. a WebSocket-only DoS on a server that registers no WS handler is likely non-applicable), and let the user opt into the too-new full fix or accept the partial one.
- **A residual advisory on a deferred/skipped package** — expected; report it as still-open with the reason (deferred by cooldown / skipped by the user), not as a failure.

Go changes:

```sh
govulncheck ./...                # if available; the GHSA should clear
```

Generated-artifact drift — these npm deps drive code generators, so a bump can move generated output. If a changed lockfile belongs to a package that feeds a generator, run its generator and check for drift (the same check CI runs):

- `mock-auth-server/**` deps → `make gen-mock-auth-oapi` (and `make gen-mock-auth-oapi-docs`), then `git status` the generated paths. Commit the regenerated artifacts as part of the patch.
- `docker/tools/**` deps (the toolbox image: redocly / orval / sqlc-adjacent tooling) → regenerate the artifacts that image produces (`make gen-*-oapi`, `make gen-query`, etc.) only if the tool whose dep changed is one of them, and check for drift.

If regeneration produces changes, include them — a security bump that silently changes generated output must not leave the tree in a state where CI's drift check fails.

### 7. Final Report

Summarize in Japanese:

- Packages patched, grouped by ecosystem / location, with the version diff and CVE ids.
- Any `overrides` added, flagged as **provisional pins to reclaim** once the parent natively ships the fix (specifier used — floor vs the exceptional exact pin, and why).
- Any `too-new` or `major-bump` entries applied (and the confirmation), or deliberately skipped.
- For each entry the cooldown caught: its triage band and the axis that drove it, so the record shows the adopt-or-wait call rested on evidence. Name any axis that came back unanswerable.
- Any `not-present` / `needs-manual` entries the user must handle another way.
- Verification results (`make lint` / `make test` / `npm audit` / `govulncheck` / drift checks).
- Regenerated artifacts, if any.

Do NOT commit, stage, or push. The user reviews the working tree and runs `/commit` manually. If they ask you to commit, note that a security patch commonly spans `docker/**` + `go.mod` + regenerated artifacts, so group them into a clear `Build:` / `Fix:` commit describing the CVEs.

## Notes

- **Targeted, not blanket.** The defining property of this skill is that it touches only advisory-named packages. If mid-run you notice an unrelated outdated dep, mention it but do not bump it here.
- **Don't over-ask.** A `clear`, non-major patch is exactly what the user invoked the skill for — apply it without a confirmation click. Reserve `AskUserQuestion` for the genuinely consequential calls: **major bumps** (breaking risk) and **too-new** opt-ins. Report those separately and clearly.
- **Respect the repo's npm cooldown; never disable it.** A `.npmrc` `min-release-age=N` is a deliberate supply-chain control. When it blocks a fix (`ETARGET ... date before <cutoff>`), the version is `blocked`/deferred — report when it clears (`publish_date + N days`), do not lower `min-release-age`, add `--before`, or otherwise route around it. A LOW triage band does not change this: triage supplies evidence, not permission, and the cooldown is enforced by npm at resolution time regardless of what the evidence says. Often the advisory's only in-cooldown fix is a fresh patch published the same day; wait it out or pick an older already-aged fixed version *only if the user approves that version*.
- **Vendoring.** If `vendor/` is present, a `go.mod` change is not done until `go mod vendor` re-syncs `vendor/modules.txt`; otherwise the build breaks with `inconsistent vendoring`.
- **Transitive vs direct.** Bumping a direct dep is preferred when a compatible direct-dep version already carries the fix; scoped `overrides` is the tool for purely-transitive deps whose parent has not yet released. A scoped override (`"parent": {"pkg": ">=<fixed> <<next-major>"}`) fixes the vulnerable nested copy without downgrading an already-patched top-level copy. State which mechanism each package used in the report.
- **Overrides are provisional debt — write a floor, then reclaim them.** An `overrides` entry is a manual, sticky pin that npm neither expires nor reminds you about, so it rots into a silent cap on a transitive dep. Two rules keep it healthy: (1) write it as a **same-major floor** (`">=<fixed> <<next-major>"`), never an exact version, so it enforces the fix as a *minimum* without freezing the dep — reserve an exact pin for when a newer in-range version is known-broken and must be held; (2) treat every override as **temporary** — once the **parent** ships a release that natively pulls a fixed version, reclaim it: bump the parent, delete the now-redundant override, `npm install --package-lock-only`, and re-run `npm audit` to confirm the fix still holds without the pin. A stale exact override can even re-introduce a vulnerability once the pinned version is itself flagged.
- **Multi-CVE packages.** A package may appear in several advisory lines (e.g. `lodash` under two CVEs) — dedup to one bump that satisfies all, and cite every CVE it resolves.
- **Lockfile-only.** `node_modules/` is not tracked; `npm install --package-lock-only` updates `package.json` + `package-lock.json` without a full install. A bump can still cause npm to re-arrange sibling copies in the lockfile — review the diff, but extra `4.17.x → 4.18.x`-style dedup churn on the same package is expected and benign.
- **Idempotency.** Re-running after a successful apply shows the packages already at the fixed version (npm audit / govulncheck clean) and makes no writes.
- The skill never auto-pushes. The user reviews, then commits and pushes manually.

## Checklist

Confirm before reporting completion:

- [ ] Advisory list parsed into (package, current, fixed candidates, CVE/GHSA, ecosystem)
- [ ] `<MIN_AGE_DAYS>` resolved (repo `.npmrc min-release-age` preferred; `7` default for Go / uncooled npm); asked only on genuine ambiguity
- [ ] Each package located (lockfile dir / `go.mod`) and classified direct vs transitive/indirect; not-present entries surfaced
- [ ] Fixed version chosen as minimal same-major; major bumps flagged as potentially breaking; downgrade guard applied
- [ ] Publish date fetched; disposition set (clear / too-new / blocked-by-cooldown)
- [ ] Every `too-new` / `blocked` entry triaged via `/supply-chain-triage` (baseline = the lockfile's current version); band carried into the summary and the `AskUserQuestion` option descriptions
- [ ] Japanese summary shown; **clear non-major applied without asking**; `AskUserQuestion` used only for major-bump / too-new; blocked entries deferred (not applied)
- [ ] npm direct via `npm install --package-lock-only`; transitive via a **scoped same-major floor** `overrides` (`">=<fixed> <<next-major>"`; exact pin only for a known-broken newer version) + `npm install --package-lock-only`; lockfile never hand-edited
- [ ] Any override recorded as provisional in the report (reclaim — bump parent, drop override, re-audit — once the parent natively ships the fix)
- [ ] Go via a single batched `go get module@ver ...` + `go mod tidy`; `go mod vendor` if the repo vendors
- [ ] `npm audit` for npm changes; `govulncheck` + build for Go changes; `make lint` / `make test` as scope warrants (major bumps verified more closely — typecheck / package tests)
- [ ] Generator drift checked for generator-feeding deps; regenerated artifacts included
- [ ] Final Japanese report: applied set, major/too-new decisions, deferred/skipped items, verification results
- [ ] After updating `SKILL.md`, re-sync `SKILL.ja.md`
- [ ] No commit / stage / push performed
