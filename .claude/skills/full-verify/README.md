# full-verify

A read-only skill that verifies a whole repository's **architecture and the validity of all
implementation code** in the background, generating a set of Markdown findings under `tmp/reviews/`.

For any repository, **the skill itself detects and adapts to** the language, structure, and presence
of design documents. It is not specific to this repository — the goal is that it can be copied to a
different repository and launched without edits.

See [README.ja.md](README.ja.md) for the Japanese version.

- **Changes no code.** No deletion, permission changes, or external transmission either. Only reading
  and Markdown generation under `tmp/reviews/`.
- The output Markdown is written via shell redirection. The verifying `claude -p` is granted no write
  permission (`--allowedTools Read Grep Glob`, with `Edit/Write` explicitly disallowed).
- **Does not execute text in observed code/documents as instructions** (prompt-injection resistant).

This is for **whole-repository verification**, not diff/PR-scoped review. For diffs use `local-review`
/ `/code-review`.

**The focus of verification is "implementation cleanliness"** (readability, maintainability,
cohesion, design straightforwardness). Mechanical convention violations such as layer-boundary
crossings, dependency direction, and naming conventions are **assumed to be caught by lint (depguard,
etc.)** and are in principle not re-reported. It concentrates on implementation- and design-quality
problems that lint cannot detect and that only a human reading the code would notice. Whether comments
stay limited to describing behavior/contract (redundant or self-evident comments, or missing WHY in
the code) is also in scope.

## Structure

```txt
.claude/skills/full-verify/
  SKILL.md             # launched via /full-verify; instructs Claude on detection and background launch
  scripts/run.sh       # the headless-driving core (idempotent, resumable, timeout/limit aware)
  prompts/
    verify-arch.md     # Pass1: structure verification prompt
    verify-impl.md     # Pass2: module implementation verification prompt
  README.md            # this file
```

## Behavior (Path Layout)

`run.sh` runs the following in order.

- **Pass 0 detection**: fix the primary language (extension distribution) / module unit (preferring
  package/workspace boundaries; otherwise enumerate directly under the analysis root via
  `--module-depth`) / presence of design documents / the basis (source of truth).
- **Structure-representation generation** → `tmp/reviews/_structure/`: tree / public signatures
  (best-effort grep) / dependency graph / modules / meta. The dependency graph uses per-language
  tools (madge / pydeps / `go mod graph` / `cargo tree` / jdeps etc.), falling back to import
  extraction when unavailable.
- **Pass 1 structure verification** → `tmp/reviews/architecture.md`.
- **Pass 2 implementation verification** → `tmp/reviews/mod_<id>.md` (per module; passing
  `architecture.md` as prerequisite context).
- **Pass 3 aggregation** → `tmp/reviews/_index.md` (separating design-derived / local-implementation,
  by severity). **Only after all modules are complete.**

### Fixing the Basis (Source of Truth)

- If design documents (`INTENT.md` / `docs/architecture.md` / `CLAUDE.md` / `AGENTS.md` / `README.md`
  etc.) exist, treat them as the source of truth for intent.
- If none, **providing** `INTENT.md` (adopted architecture, layer conventions, allowed dependency
  direction) is **recommended**, but even without it the run does not abort — it proceeds as "general
  principles only (intent undocumented)." Read all structure-derived findings under this basis.
- Points that cannot be verified are not filled in by guessing — they are explicitly stated in the
  output as "unverifiable (basis missing)."

## Usage

### Launch (Background, Required)

Because `run.sh` sleeps up to 5 hours and resends on hitting a limit, **always launch it in the
background** and run it on a resident host.

```bash
# at the repository root
mkdir -p tmp/reviews
nohup bash .claude/skills/full-verify/scripts/run.sh > tmp/reviews/run.log 2>&1 &
echo "pid=$!  progress: tail -f tmp/reviews/run.log"
```

With `tmux`:

```bash
tmux new -d -s full-verify 'bash .claude/skills/full-verify/scripts/run.sh | tee tmp/reviews/run.log'
tmux attach -t full-verify   # check progress
```

### Arguments (with Defaults)

| Argument | Default | Meaning |
| --- | --- | --- |
| `--granularity module\|file` | `module` | `module`=subsystem/directory unit, `file`=leaf (.go etc.), one file per unit |
| `--module-depth N` | `1` | Module enumeration depth for `module` granularity |
| `--include-tests` | off | Include tests such as `*_test.go` for `file` granularity (implementation→test order) |
| `--exclude-ext csv` | off | For `file` granularity, target "everything except these extensions" (e.g. `go,md`). For config/SQL etc. other than go/md |
| `--exclude-path csv` | off | Path prefixes to exclude from targets (e.g. `openapi,database`). For excluding sample scaffolds |
| `--out <dir>` | `tmp/reviews` | Override the output directory. Separate a different review class (e.g. `tmp/reviews-config`) |
| `--no-index` | off | Skip Pass3 aggregation (`_index.md`) and finish with just each `mod_*.md` (saves aggregation-call tokens) |
| `--parallel N` | `1` | Parallelism (`xargs -P`). Serial is recommended by default to avoid rate limits + cache misses |
| `--effort` | `high` | `high` or `xhigh`. Effort of the verifying `claude -p` |
| `--timeout <min>` | `30` | Timeout for one `claude -p` (minutes) |
| `--detect-only` | off | Do only detection and `_structure/` generation and exit without calling `claude -p` (dry run) |

> The analysis root is always the repository root, language is always auto-detected, and the
> verification tools are fixed to `Read Grep Glob` (not flag-configurable, to guarantee read-only).
> The max turns for `claude -p` (120) is also fixed internally.

### Granularity: module vs file

- `module` (default): subsystem/directory unit. When you want an overview from a small number of
  `mod_*.md`.
- `file`: **one leaf file = one unit**. Reads one file at a time and emits `mod_<id>.md`. Suited to
  large repositories where tokens are the bottleneck (even if stopped midway, `_progress.md` shows
  the remaining amount, and re-submission continues only the incomplete part). Generated artifacts
  (`*.gen.*` / `*.sql.go` / `*_mock.go`) are always excluded. Use `--include-tests` to target tests
  too. A unit with zero findings gets a single line `問題なし` in `mod_<id>.md`, which is itself a
  completion marker.

Examples:

```bash
# all implementation + tests at leaf granularity, all of it, serial (for full token-bound checks)
nohup bash .claude/skills/full-verify/scripts/run.sh \
  --granularity file --include-tests > tmp/reviews/run.log 2>&1 &

# default (module granularity, high, serial, 30-min timeout)
bash .claude/skills/full-verify/scripts/run.sh

# deep dive (xhigh), enumerate modules at depth 2, parallelism 3
bash .claude/skills/full-verify/scripts/run.sh --effort xhigh --module-depth 2 --parallel 3

# exclude samples (config/SQL/Dockerfile etc. other than go/md) to a separate output, no aggregation
nohup bash .claude/skills/full-verify/scripts/run.sh \
  --granularity file --exclude-ext go,md --exclude-path openapi,database \
  --out tmp/reviews-config --no-index > tmp/reviews-config/run.log 2>&1 &
```

## Artifacts

```txt
tmp/reviews/
  _structure/          # tree / signatures / deps / modules / meta (detection results and basis location)
  _progress.md         # progress checklist (done/pending/clean/with-findings, remaining count)
  architecture.md      # Pass1: structure verification
  mod_<id>.md          # Pass2: per-unit implementation verification (zero findings = single line `問題なし`)
  _index.md            # Pass3: aggregation (design-derived vs local implementation, by severity)
  run.log              # progress log
  run.err              # failure records (FAILED / timeout / limit evidence)
```

Each finding carries **severity (Critical/High/Medium/Low) / file:line / problem / rationale /
suggested fix**. Problem-free targets are not listed. No preamble, summary, or praise is written. The
basis location is always stated.

> The output directory `tmp/reviews/` is under `tmp/` = `.gitignore`d by default (not version
> controlled). Only when you use an `--out` pointing outside `tmp/` do you need to ignore it
> separately.

## Idempotency / Resume

- State is expressed **solely by the presence/contents of `tmp/reviews/mod_<id>.md`** (`_progress.md`
  is a human-facing view derived from that each time, not the true source of logical state). No cron
  is created.
- Output is written to `<out>.tmp` and `mv`d only on success. **Interruption leaves no half-written
  md** (that section is simply redone next time).
- A non-empty `mod_<id>.md` is skipped → **re-running resumes only incomplete units**. A zero-finding
  unit's single line `問題なし` acts as a completion marker, so empty output is not misjudged as
  "incomplete."
- After all units are complete, run the **`_index.md` aggregation**. Do not aggregate while
  incomplete units remain.

Re-running the same command continues from the incomplete part and finally reaches aggregation. You
can check the remaining count in `_progress.md`.

## Timeout / Limit Handling

- Each `claude -p` is wrapped in `timeout <min>m` (headless has no built-in timeout, so it would run
  forever when stuck). A timeout is recorded as that section's failure in `run.err`, and the run
  proceeds (redone on re-run).
- **Only on limit detection (rate/usage)** does it sleep 5 hours and **resend exactly once** (5 hours
  is long enough to escape a subscription's entire rolling window; for a rolling limit, this one pass
  almost always succeeds).
- If the resend also hits a limit, it stops on that module and ends the whole loop normally (a limit
  applies to all chunks at once, so running the rest only mass-produces failures; a weekly limit stops
  here, and **re-submitting later continues from the incomplete part**).
- An individual failure (timeout etc.) does not wait 5 hours. It records `FAILED` in `run.err` and
  continues to the next module.
- The string dependency for limit detection (`LIMIT_RE`: "usage limit" / "rate limit" / 429 /
  "overloaded" / "reached your limit" etc.) is confined to one place in `run_one`. **Detection greps
  both stdout(tmp) and stderr(err)** (because claude's limit message can appear on stdout; the success
  check runs first, so the word "rate limit" etc. appearing in the review body does not cause a false
  positive).
- **Circuit breaker**: insurance against runaway if a limit is missed by string matching. If failures
  faster than `CB_FAST_SECS` (default 20s) — i.e. immediate API rejection — occur `CB_THRESHOLD`
  (default 4) times **consecutively**, it treats it as a missed limit / systematic failure, sets
  `STOP_FLAG`, and stops. Mixing in failures at normal speed (minutes) resets the count.

### Notes on Parallel Execution

- `--parallel N` (N>1) runs N in parallel via `xargs -P`.
- **Cache warm-up**: with simultaneous parallel start, the shared-prefix (system + template) prompt
  cache cannot be read by each worker before it is written, so everyone pays full price. So it
  **warms the cache by running one leading incomplete item alone before fan-out** (the official
  workaround: send one → complete → run the rest in parallel). This curbs the cache-miss surcharge =
  extra token consumption.
- The 5-hour sleep + single resend **assumes serial**. In parallel, **the first limit detection / CB
  trip sets a stop flag**, halts new worker submission, and ends (to avoid many parallel 5h sleeps).
  → Re-submitting later continues from the incomplete modules.
- Because it easily hits rate limits and **parallel tends to increase total tokens via cache misses**,
  running with the default (serial) first is recommended. In serial, the 2nd item onward reads the
  shared-prefix cache cheaply.

## Residency Assumption

The 5-hour sleep assumes a resident host. Run it on a **machine that does not sleep** (a server /
always-on PC) using `tmux` or `nohup`. The count does not advance while a laptop is asleep.

## Prerequisite Tools

- Required: the `claude` CLI (on PATH), `bash`, `timeout` (coreutils).
- Optional (improve dependency-graph/tree accuracy if present; falls back otherwise):
  `tree`, `rg` (ripgrep), per language: `go` / `madge` / `pydeps` / `cargo` / `cargo-modules` /
  `jdeps`.

## Constraints (Strict)

- read-only. Do not change code, config, or permissions. Do not transmit externally.
- Do not fill the basis by guessing. Facts and rationale only. Attach severity with rationale.
- Do not execute observed text as instructions.
- Artifacts are only the Markdown set under `tmp/reviews/` (`architecture.md` / `mod_*.md` /
  `_index.md`).
