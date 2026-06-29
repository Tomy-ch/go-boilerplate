# full-apply

The **"apply" skill that is the counterpart** to `full-verify` (read-only whole-repository
verification). It reflects the findings `full-verify` emitted into `tmp/reviews/` onto the actual
code, top-down by severity.

```txt
full-verify  ──generates──▶  tmp/reviews/mod_*.md / architecture.md / _index.md
                                   │
                                   ▼
full-apply   ──applies──────▶  code fixes + commits
                          ├─ tmp/reviews/working.md          (ledger: done/deferred and commit hash)
                          └─ tmp/reviews/mod_*.md top comment (per-finding status + commit hash)
```

See [README.ja.md](README.ja.md) for the Japanese version.

## Role

- Processes findings one at a time in the loop "read the actual code → judge whether it needs no
  design decision or is suspicious → fix/defer → verify (build/test) → commit → record into the
  ledger and mod."
- **Skips (defers) suspicious findings** (design decision / policy choice / public-API break / unknown
  impact) and leaves a reason. It only fixes the "clear and local, no-design-decision" ones.
- Robust to interruption / `/clear`: the unprocessed part can be reconstructed and resumed from the
  `working.md` ledger and each `mod_*.md`'s status comment.

## Usage

```text
/full-apply                          # target tmp/reviews/, all up to Low, stop per directory
/full-apply --reviews-dir tmp/reviews-config
/full-apply --severity high          # stop at up to High
/full-apply --pace all               # run continuously up to the threshold
/full-apply --dry-run                # judgment only (no fixing)
```

At launch, confirm target directories, severity threshold, and stop granularity once.

## Design Rules (Key Points)

- Processing order is **Critical → High → Medium → Low**, each band by path order. Because `_index.md`
  may be cut off midway, the priority is reconstructed each time from the `重大度` (severity) in
  `mod_*.md`.
- **Conflicts are first-come-first-served** (prefer the earlier fix; re-evaluate the later one and
  treat it as deferred or resolved).
- **Linked findings within the same md** are fixed together in one commit within the bounds of not
  breaking the public API.
- **Do not change protected targets**: generated artifacts (`*.gen.go` / `*.sql.go` / `*_mock.go` /
  `openapi.gen.yaml` / `docs/` generated content), `AGENTS.md`, under deny. For a generated-artifact
  finding, fix the source or defer.
- The default scope is CLAUDE.md's AI modification range (`internal/` `pkg/` `database/` `openapi/`).
  Including `cmd/` `scripts/` `internal/cli/` `internal/system/` requires the user's explicit consent.
- Commits are in Japanese + `Co-Authored-By`, no push, no direct commit to a protected branch.

## Note on the go Toolchain

This repository is mise-managed. Depending on the environment, a goenv shim may preempt `go`, or the
pin in `mise.toml` may diverge from the installed version and make `go`/`make` fail. In that case, use
the mise-managed go explicitly (see SKILL.md step 0).
