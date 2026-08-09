---
name: impl-issue
description: >-
  Drive a GitHub issue from environment setup to a merged PR as a semi-automatic pipeline that stops at every decision the human owns. Use whenever the user hands over an issue URL or number to be worked end-to-end ("この issue やって", "wt 上で解決しよう", "着手して PR まで"), or asks to resume such a run. It owns three things — progress orchestration, reconciling the approved plan against what was actually built, and mechanically detecting the moments needing a human call — and no implementation judgment: the work is delegated to `commit` / `submit-pr` / `impl-review` / `test-review`, and design decisions are surfaced, never taken. It sets up an isolated worktree and DB slot, has a different model draft a written plan the user approves before coding, then watches five mechanical trip-wires so drift becomes visible instead of silent. Runtime verification (`make serve` + curl + LGTM traces) runs after the PR is opened and gates the merge; green CI is not a substitute. Three modes are confirmed once: review mode, issue mode, and flow mode. Do NOT use for a change with no issue behind it (`commit` + `submit-pr`), for reviewing an existing diff (`impl-review` / `test-review`), or for authoring skills (`manage-skill`).
argument-hint: '<issue-url-or-number> [--review-mode=all|harmful|issues] [--issue-mode=search|file] [--flow=interactive|delegated]'
---

# Impl Issue

Semi-automatic issue → PR pipeline. The machine handles progression, bookkeeping, and detection; the
human keeps every judgment call. The point is not speed — it is that a long autonomous run stops
being a black box, because each place where the work departed from the approved plan surfaces when it
happens rather than at the end.

This file is self-contained on purpose. Anyone should be able to reproduce the same run on a
different machine from this file alone, so the concrete commands live here rather than in local
notes. Repository detail that genuinely does drift stays behind pointers: `.makefiles/README.md`
(full target registry), `docs/maintenance/db-worktree-pool.md` (slot pool), `docs/development-flow.md`
(per-change-type flows).

A Japanese reference translation lives at `SKILL.ja.md` in this directory (for human reference only;
not loaded as a skill).

## When to Use

- The user hands over an issue URL / number and wants it taken to a merged PR.
- The user asks to resume a run that stopped at a decision point.

Do NOT use it for a change with no issue behind it (`commit` + `submit-pr` directly), for reviewing an
existing diff (`impl-review` / `test-review`), or for authoring skills (`manage-skill`).

## What this skill does NOT do

It holds no implementation judgment. It never decides which design to adopt, whether a reviewer is
right, or whether a finding deserves an issue. It routes those to the user and records the answer.

| Work | Owner |
| --- | --- |
| Commit splitting and execution | `commit` |
| Push + PR create/update | `submit-pr` |
| Adversarial code review | `impl-review` |
| Test-quality review | `test-review` (chained by `impl-review`) |
| The implementation itself | you, following the approved plan |

## AI Modification Scope

`AGENTS.md` confines AI edits to `internal/` / `pkg/` / `database/` / `openapi/` and treats everything
else — `.github/workflows/`, `docker/`, `scripts/`, `docs/`, `.makefiles/`, root dotfiles — as out of
scope. **Invoking this skill is the explicit user instruction that relaxes that**, because this skill
is issue-generic: the issue decides the surface, and an issue about CI, tooling, container images, or
documentation cannot be resolved inside the four default directories. This is a documented,
non-loophole exception per the "Skills must not be a loophole" clause in `AGENTS.md`.

The relaxation is bounded, and the bound is the plan:

- The Step 3 plan's **Files to touch** section is the permitted surface. A sensitive path outside the
  four default directories must appear there **before** it is edited, named explicitly rather than
  implied by a glob.
- Say so when presenting the plan. The user approves the sensitive paths knowingly, not by discovering
  them in the diff — a plan that quietly widens the scope is the failure this clause exists to prevent.
- Reaching a sensitive path the plan does not list is trip-wire 1 (Step 4). Stop and ask; do not widen
  the surface and report it afterwards.

Hard-protected even during this skill (never touch, regardless of what the issue asks):

- `AGENTS.md` / `CLAUDE.md`
- Generated files: `**/*.gen.go`, `*.sql.go`, `*_mock.go`, `**/openapi.gen.yaml`, and generated content
  under `docs/` (`docs/openapi/**`, `docs/coverage/**`, `docs/db-schema/**`, `docs/godoc/**`,
  `docs/portal/docs.json`, `docs/portal/guides/**`). Regenerating through a `make` target is fine;
  hand-editing is not.
- Anything under `permissions.deny` in `.claude/settings.json`
- Existing files under `database/migrations/**` (new migration files only)

## Step 0 — Confirm the three modes (one `AskUserQuestion`)

Ask once, before anything else, in a single call. Defaults are marked; the user's choice always wins.

**Review mode** — what happens to a review finding.

| Mode | Confirmed finding | Everything else |
| --- | --- | --- |
| `all` | Apply, even if the change is large | — |
| `harmful` *(default)* | Apply only what is clearly harmful within the change's scope | Route to issue mode |
| `issues` | Apply nothing | Route to issue mode |

**Issue mode** — how an unrelated finding becomes tracked.

| Mode | Behavior |
| --- | --- |
| `search` *(default)* | Search existing issues first; on a duplicate, comment there instead of filing |
| `file` | File without searching |

`issues` × `file` produces the most new issues of any combination. Before executing it, show the count
and confirm — a review easily yields a dozen findings, and a dozen new issues is itself noise.

**Flow mode** — what a trip-wire does.

| Mode | Behavior |
| --- | --- |
| `interactive` *(default)* | Stop and ask |
| `delegated` | Record the call and continue; surface all recorded calls in a PR comment at the end |

`delegated` is for when the user has handed over full authority *and* will be away — stopping for
permission nobody is there to grant kills the run. It is not a speed setting.

## Step 1 — Kickoff

```bash
printf '\033]0;%s\007' "<issue-number>-<slug>"   # label the window so parallel runs stay distinguishable
gh issue view <n> --json number,title,body,labels,state,comments
```

**Compare the issue against the actual base before writing anything.** An issue body is a snapshot of
the repo as it was when someone wrote it; line numbers, "X does not exist yet", and "Y has no consumer"
go stale. Verify each factual claim against the base you are about to branch from.

Then post a kickoff comment recording branch name, base commit, isolation method, and — most
importantly — **every discrepancy found above**. This marks the issue as taken and gets the
corrections to the user while they are still cheap.

```bash
gh issue comment <n> --body-file <file>
```

## Step 2 — Secure the environment

Do this before any code is touched, so nothing lands in a shared checkout. Which half applies depends
on whether the worktree exists yet — setting one up and resuming into one are different operations,
and running the first against an already-live worktree destroys work.

### Initial setup — no worktree yet

```bash
# 1. Resolve the active release line off origin's live state.
BASE=$(make -s base-branch)
test -n "$BASE" || { echo "ベースブランチを解決できませんでした"; exit 1; }

# 2. Branch from current origin, not a stale local ref.
git fetch origin "$BASE"
git worktree add -b feature/<n>-<slug> ../go-boilerplate.worktrees/<n>-<slug> "origin/$BASE"

# 3. Lease a DB slot: own databases (wt<N>_local / wt<N>_test), API port 8080+N, mock-auth 2010+N.
cd ../go-boilerplate.worktrees/<n>-<slug> && make slot-acquire

# 4. A fresh worktree has no vendor/ and air builds with --mod=vendor, so serve would fail without this.
go mod vendor
```

`make base-branch` reads `origin`'s live state. Use nothing else for this: the local
`refs/remotes/origin/HEAD` is set once at clone time and `git fetch` never updates it, the GitHub
default branch stays on an earlier release line, and the harness's own "Main branch" line reports that
stale local symref. All three answer without warning, and branching from a generation-old base is not
visible until a subagent reports that files everyone expects are missing — by which point the work on
that branch is wasted.

If `slot-acquire` reports failure, run `make slot-status` before retrying — the lease often succeeded
even when the command errored.

**Never release the slot on your own, and do not offer to during cleanup.** A slot is cheap to hold
(the lease is reclaimed automatically once stale) and expensive to lose mid-task; only the user knows
when the work is really over.

If the user's instruction named a release version other than the resolved one, ask before branching —
a deliberate backport target is the one case the resolver cannot know about.

### Resuming into an existing worktree

Setup already happened. Observe it and report what you found; do not re-run any of it.

The session-start environment line (`[agent-env] checkout=… branch=… vendor=… db-slot=…`) already
carries the three facts that matter — which checkout this is, whether `vendor/` exists, and which DB
slot is held. When it is present, state those back before continuing rather than re-deriving them.

Do not depend on it being there. A hook can be absent, disabled, or attached to a harness that never
ran it, and the fallback has to be inspection rather than repair. Every command below only reads:

```bash
git rev-parse --show-toplevel
test -d vendor && echo 'vendor: present' || echo 'vendor: absent'
cat .gobp-db-slot 2>/dev/null || echo 'slot: none'
```

A missing or malformed slot is a fact to report, not a fault to fix on the spot. Lease one with
`make slot-acquire` immediately before DB-backed work actually begins — the first `make test`,
`make serve`, or `psql` — and not before. `go mod vendor` is the same: run it when `vendor/` is absent
and a build is imminent, not as a resume ritual.

**Never acquire a slot or reinitialize a database because a session resumed.** `slot-acquire` and the
`db-*-reinit` targets rebuild the slot's `wt<N>_local` / `wt<N>_test` databases from scratch, so a
reflexive resume-time acquire destroys the state belonging to the very run it is resuming — seeded
fixtures, a half-applied migration, the data a failure was about to be diagnosed from.

## Step 3 — Plan, then wait

Have a **different model** draft the plan — a second model catches what the implementer's own blind
spots would otherwise carry straight into the code. Give it the issue, your Step 1 corrections, and
the paths you have already read. Tell it to verify your summary rather than trust it.

The plan is a written artifact, not a chat message, because Step 5 compares against it mechanically.
Write it under the repo's gitignored `tmp/` (it may be a symlink to a directory outside the repo if
the operator prefers). It must contain:

| Section | Why it is required |
| --- | --- |
| Files to touch | Step 5 diffs this against `git diff --name-only` |
| Per-step deliverables | Lets a partially-finished run be resumed or handed over |
| Chosen options **and rejected ones, with reasons** | Trip-wire 2 fires when a rejected option is later adopted |
| Gate table | Fixes at plan time whether runtime verification is required, so it cannot be quietly dropped |

Present the plan and **wait for approval. Do not implement before it.**

## Step 4 — Implement, watching five trip-wires

Follow the approved plan. These triggers are deliberately mechanical — relying on you to *notice* that
a decision was significant is exactly how drift goes unreported.

| # | Trip-wire | Why it is a human call |
| --- | --- | --- |
| 1 | Touching a file the plan does not list | Scope grew; the user approved a different shape |
| 2 | Choosing an option the plan rejected, or a third one | The rejection had a reason; overriding it silently discards that reasoning |
| 3 | A lint/CI failure rooted in an architecture rule (`interfacebloat`, `gocognit`, `depguard`, architest, …) | These are not formatting — satisfying them changes the design |
| 4 | Rejecting a reviewer's finding, or applying a different fix than proposed | A finding can be correct while its proposed fix is harmful; that judgment is not yours alone |
| 5 | Skipping any gate | See Step 6 |

On a trip-wire: `interactive` → stop and present the situation with your recommendation.
`delegated` → record it and continue, then surface every recorded call in one PR comment at the end.

### When code generation is blocked

The generation make targets wrap `docker compose run … make <target>-ci`, and the `-ci` halves run on
the host with mise-installed tools. When the container runtime is unavailable, call them directly:

```bash
make merge-dml-ci work-dir="."     # DML concatenation
make sqlc-generate-ci              # sqlc
make gen-bundle-oapi-ci            # OpenAPI bundle
make gen-api-docs-ci               # docs/openapi/index.html
cd <pkg> && mockgen -source=<f>.go -destination=mock/mock_<f>.go.gen.go -package=mock_<pkg>
```

Only `make dump-schema` truly needs the container, and only when a migration was added.

Two traps: `merge-dml-ci` runs `go run ./cmd/`, so adding a Repository method before its query exists
deadlocks the build — stub the implementation for the duration of generation, then restore (back the
file up with `cp` first). And the embedded-spec generator's `//go:generate` line points at a container
path, so `go generate` cannot run it; invoke it with the real path:

```bash
cd internal/controller/httpstack/oapi/validator \
  && oapi-codegen --package=gen --generate=spec -o ./gen/validate.gen.go <repo>/openapi/openapi.gen.yaml
```

Changing an OpenAPI description alone still moves three artifacts: the bundle, `docs/openapi/index.html`,
and that embedded spec. Miss one and CI's generate checks fail.

## Step 5 — Reconcile the plan against reality

Run this before the gates. Compare:

- `git diff --name-only` against the plan's file list — report additions and untouched entries.
- Options actually taken against the plan's chosen/rejected lists.
- Gate table entries against what you actually ran.

Present the deltas. A long run drifts for good reasons; the problem is drift the user never saw. If
nothing drifted, say so in one line and move on.

## Step 6 — Local gates

`make fix`, then `make lint` / `make test`. When many worktrees are active these may be delegated to
CI — that is a documented trade-off — but **say in the PR that they were not run locally**. Silence
reads as "verified".

Runtime verification is deliberately *not* here. It belongs after the PR exists (Step 8), so CI runs
in parallel with it instead of after it.

## Step 7 — Review

Run `impl-review` (which chains `test-review`). Handle findings per the review mode from Step 0.

Auto-application is confined to things whose correctness is machine-checkable: formatting, lint fixes,
comment-quality findings, regenerated artifacts. **A fix that changes the design is always a decision
point**, even under review mode `all` — `all` authorizes a large rewrite, not an unreviewed one. This
mirrors why `impl-review` keeps its five code lenses report-only.

Then present every trip-wire and deferred judgment together, in one place. Batching beats trickling:
the user sees the shape of the whole run at once.

## Step 8 — PR, then runtime verification, then merge

Open the PR first via `submit-pr`, so CI starts while you verify locally.

### Runtime verification — the merge gate

Exercise the real HTTP path against the running system. No mode relaxes this.

```bash
make serve                                    # API on 8080+N, mock-auth on 2010+N

TOKEN=$(curl -s -X POST http://localhost:201N/bypass/token \
  -H 'Content-Type: application/json' \
  -d '{"subject":"user-john-doe","profile":"valid"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["access_token"])')

curl -s -o /dev/null -w '%{http_code}\n' -H "Authorization: Bearer $TOKEN" http://localhost:808N/v1/...
```

The token subject must be the `user_identities.subject` string (`user-john-doe` form), not a user
UUID — the seeded UUID rows belong to a different issuer than the one the slot's port produces, so a
UUID yields a confusing 401. Resolve real subjects from the DB when unsure:

```bash
docker exec gobp-shared-database-1 psql -U postgres -d wt<N>_local -c \
  "select ui.subject, r.name from user_identities ui
     left join user_roles ur on ur.user_id = ui.user_id
     left join roles r on r.id = ur.role_id
   where ui.issuer = 'http://localhost:201N';"
```

Check the happy path, the error paths the change introduces, and — for a protected operation — that
omitting the token gives 401. Then **read the traces in the LGTM stack and confirm the request took
the path you expect** (controller → usecase → infrastructure, with the SQL you intended). A response
code alone does not prove the request reached the layer you changed; a wrong-but-plausible route
produces the right status for the wrong reason.

**Green CI is not a substitute for this.** A previous run merged a change whose documented API
contract was wrong — it claimed 409 where the running system returns 401, because authentication
rejects the caller before the usecase is ever reached. Five review lenses and 29 CI checks passed,
because every one of them was static analysis or a test that stopped at the database layer. One real
HTTP request exposed it immediately.

When runtime verification cannot run at all, there are two honest options and no third:

1. Do not merge yet.
2. Add an integration test driving the same HTTP path, and merge on that.

Say plainly which one you took.

### Merge

Wait for CI without burning the session on a foreground sleep loop — run a background command that
exits once the checks settle, so one notification arrives:

```bash
until [ "$(gh pr checks <n> --json bucket --jq '[.[]|select(.bucket=="pending")]|length')" = "0" ]; do sleep 30; done
gh pr checks <n>
```

Then `gh pr merge <n> --merge`.

## Step 9 — Close out

Close the issue **manually** — auto-closing keywords do not fire when the PR targets a release branch
rather than the default branch:

```bash
gh issue comment <n> --body-file <handover> && gh issue close <n>
```

The handover comment covers what was decided, what surprised you relative to the plan, and what was
deliberately left undone.

Route findings that fall outside the change to the tracker per the issue mode. Under `search`, prefer
a follow-up comment on an existing issue over a new one — the issue count is itself a cost, and a
duplicate buries the original.

**Verify a finding against the running system before filing it.** A finding derived purely from
reading code can be wrong in a way static review cannot catch: an earlier run filed an issue claiming
withdrawn users could still call several endpoints, when middleware in fact rejected them all long
before the code that had been inspected. That issue had to be closed as not-planned. Step 8's runtime
stage is usually enough to check.

Finally, record in a PR comment any call not already visible in a commit message or the PR
description. In `delegated` mode this is where the recorded trip-wires land.

## Delegating without double-asking

Sub-skills ask their own questions. Since this skill already settled them with the user, pass the
answers as a payload so the sub-skill skips its own gate — the way `impl-review` hands `scope` /
`base_ref` / `reviewer_model` / `skip_verifier` to `test-review`.

| Sub-skill | Pass through | Suppresses |
| --- | --- | --- |
| `commit` | The grouping you already presented | Its grouping-approval question |
| `submit-pr` | That review already ran; the flow-mode push decision | Its Phase 0 review prompt and push confirmation |
| `impl-review` | Scope, reviewer model, test-delegation choice | Its Step 0 |

Asking the user the same thing twice trains them to approve without reading, which defeats the
decision points this skill exists to create.

## Do / Do NOT

- ✅ Secure the worktree and slot before touching code.
- ✅ On resume, inspect the worktree and report what you found — path, `vendor/`, slot.
- ✅ Verify the issue's claims against the actual base, and put the discrepancies in the kickoff comment.
- ✅ Get the plan approved before implementing, and keep it as a file so Step 5 can diff against it.
- ✅ Treat the five trip-wires as mechanical triggers, not as things to notice.
- ✅ Say explicitly which gates ran and which did not.
- ✅ Read the traces, not just the status code.
- ✅ Verify a finding at runtime before filing an issue for it.
- ❌ Merge a change to implementation code that has never been exercised over HTTP.
- ❌ Present green CI as runtime verification.
- ❌ Auto-apply a fix that changes the design, in any mode.
- ❌ File an issue without checking for an existing one, unless issue mode says to.
- ❌ Release the DB slot, or ask about releasing it, unprompted.
- ❌ Acquire a slot or reinitialize a database merely because a session resumed.
- ❌ Poll CI in a foreground sleep loop.

## Checklist

- [ ] Modes confirmed in one `AskUserQuestion`.
- [ ] Kickoff comment posted, including issue-vs-base discrepancies.
- [ ] Environment secured on the right half of Step 2: a new worktree created from a freshly fetched
      base with a slot leased and `go mod vendor` run — or an existing one observed and reported,
      with no slot acquired and no database reinitialized just because the session resumed.
- [ ] Plan drafted by a different model, all four sections present, approved before implementation.
- [ ] Trip-wires handled per flow mode; nothing silently absorbed.
- [ ] Plan reconciled against the actual diff.
- [ ] Local gates run, or their delegation to CI stated in the PR.
- [ ] Review run; auto-application confined to machine-checkable fixes.
- [ ] Decision points presented together.
- [ ] PR opened, then runtime verification (curl + traces) completed — or its absence stated together
      with which of the two options was taken — before merging.
- [ ] Issue closed manually with a handover comment; unrelated findings routed per issue mode, each
      verified before filing; remaining judgment calls recorded in a PR comment.
