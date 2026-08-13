---
name: comment-sweep
description: >-
  Sweep the EXISTING STOCK of source-code comments in a chosen scope and decide, per comment, whether its content is in the right place — the jurisdiction question that no other reviewer asks. Where `comment-reviewer` (inside `impl-review`) judges comments on a diff and can only answer 削除 or 書換, this skill runs over accumulated code and adds the missing verdict: 移設 (relocate a design rationale out of the comment and into `docs/adr/` / `docs/design/**` / `docs/spec/**` / a package README, leaving only the operative residue plus a link — while refusing the two classic misroutes, since a library's specific behavior stays in the code and business knowledge goes to spec, never to an ADR). Use it whenever comments feel bloated, verbose, over-explained, or essay-like even though each line is individually true; whenever a doc comment has grown into a design argument, threat-model analysis, or rejected-alternative discussion; for a periodic hygiene sweep of a package / layer / whole repo; before a large PR or a fork/template cut where accumulated commentary would burden downstream readers; and when someone asks 「コメントが長すぎる」「コメントを整理して」「この Why はコードに置くべきか」「コメントを ADR に移したい」. It reads the Comment Rules in `docs/rules.md` and the destination table in `docs/adr/README.md` at runtime as the single source of truth and hardcodes no policy, fans out read-only auditors per package, then drives a per-item approval loop and performs the code + destination-document writes itself so a relocated rationale never loses its home. Do NOT use it to review comments on a change you just wrote (`impl-review` with `comment-reviewer` owns diff scope), to judge README / docs prose quality (`doc-reviewer`), to fix README↔code structural drift (`back-prop` / `sync-readme`), or to delete `// Name は、〜です。` field comments — that repo convention is deliberately preserved and out of scope here.
---

# Comment Sweep

Judge accumulated comments on one question the existing reviewers cannot ask: **does this content
belong here?**

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory
(not loaded as a skill; for human reference only).

## When to Use

- Comments in a package read as bloated / essay-like even though nothing in them is wrong.
- A doc comment has grown into a design argument (rejected alternatives, threat model, architecture policy).
- Periodic hygiene sweep of a package, a layer, or the repo.
- Before a fork / template cut, where accumulated commentary becomes downstream reading burden.

Do NOT use for:

- Comments on a change you just wrote — `impl-review` fans out `comment-reviewer` for diff scope.
- README / `docs/**` prose quality — `doc-reviewer`.
- README↔code structural drift — `back-prop` / `sync-readme`.

## Why this skill exists (read this before judging anything)

The repo already forbids How-narration, 経緯, and restatement, and `comment-reviewer` already detects
them. Comments kept growing anyway. Two structural reasons, and understanding them is what makes this
skill work:

1. **The argument to keep always beats the argument to cut.** "This Why is non-obvious and
   verifiable" is a concrete claim with explicit permission in `docs/rules.md`. "Volume is a cost"
   is an abstract one. Concrete beats abstract every time, so every judgment call resolves toward
   keeping, and the stock only grows. You will not win by re-arguing volume — do not try.
2. **Review is diff-scoped.** `impl-review` never re-examines what is already there. There is a path
   in and no path out.

So this skill does not re-litigate whether a Why is good. It asks a **different, equally concrete
question** — one that can actually beat "but it's non-obvious":

> **If someone reversed this decision, which document would they be obliged to update?**

Write it there, link to it from the code, and keep in the comment only the **operative residue**: the
one or two sentences a person editing *this* declaration must not violate. When the honest answer is
"no document — the constraint exists only at this call site", the code **is** the jurisdiction and
the comment stays in full. This is a relocation skill, not a deletion skill.

## Authoritative sources — read at runtime, hardcode nothing

| Question | Source of truth |
| --- | --- |
| What may a comment contain; the jurisdiction clause | **Comment Rules** in `docs/rules.md` |
| Where does relocated content belong (decision / exclusion / rule / inventory) | the *What belongs here* table in `docs/adr/README.md` |
| What a subsystem design reference is for | `docs/design/README.md` |
| Package-level scope | the nearest ancestor `README.md` |

Read these at the start of every run and apply them verbatim. They may have changed; a remembered
version is not good enough. If anything in this file disagrees with `docs/rules.md`, `docs/rules.md`
wins.

### Relocating is not dumping — each destination has an entry bar

A relocation only helps if the prose lands where that *kind* of knowledge is owned. A document that
accepts everything answers nothing, and `docs/adr/` is the one most at risk of becoming the default
bucket, because from inside a comment almost anything reads as "design rationale". Two misroutes to
refuse outright:

- **A library's or an API's specific behavior** (this driver returns X on Y, this SDK reads that env
  var) is not a choice among alternatives — it is a property of the thing being called, and it
  changes when the dependency is upgraded. Its home is the comment at the call site. Verdict: 維持.
- **Business / domain knowledge** (what a rule means, why a status transitions this way) belongs to
  `docs/spec/**`, where the behavior is specified and kept current. Never route it to an ADR.

`docs/adr/` takes only a **choice among alternatives with lasting consequences** or a deliberate
exclusion. When a candidate fits no destination, that is evidence the code was the right place all
along — return 維持 rather than forcing a home. Proposing a bad destination is worse than proposing
nothing, because a wrong move is much harder to undo than a comment left alone.

## Step 0 — Confirm scope

`AskUserQuestion`, one question:

- 「comment-sweep の対象スコープを選んでください」
  - 「指定パス配下（パッケージ / ディレクトリを続けて指定）」
  - 「ベースブランチとの diff で変更されたファイル」
  - 「レイヤ全体（`internal/domain` / `usecase` / `controller` / `infrastructure` / `pkg` から選択）」
  - 「キャンセル」

Stock scope is the point of this skill — diff scope exists only so it can be chained after a large
refactor, and `comment-reviewer` remains the better tool there. If the user picks diff scope, say so
in one line and continue.

## Step 1 — Resolve targets

Exclude generated files and tests; they are not this skill's business:

```sh
find <scope> -name '*.go' \
  ! -name '*.gen.go' ! -name '*.sql.go' ! -name '*_mock.go' ! -name '*_test.go'
```

Non-Go sources (shell, Dockerfile, Makefile, SQL, YAML, `.mjs`) are in scope for the same content
standard — `docs/rules.md` says so explicitly, and they are higher-risk because `revive` does not
see them. Include them when they fall under the chosen scope.

Rank by comment volume so the sweep starts where the payoff is, and tell the user the ranking:

```sh
for f in <files>; do
  printf '%s %s\n' "$(grep -c '^[[:space:]]*//' "$f")" "$f"
done | sort -rn | head -30
```

Group the resolved files by package directory — that is the fan-out unit, because jurisdiction is a
per-package judgment (a package's own README is one of the candidate destinations).

## Step 2 — Fan out read-only auditors in parallel

For each package group, spawn one auditor via the **Agent tool** (`subagent_type: general-purpose`),
all in a **single message with multiple tool calls** so they run concurrently. Give each:

- the package directory and its resolved file list
- the instruction to read `references/audit-prompt.md` in this skill's directory and follow it verbatim
- the repo-root-relative paths of the authoritative sources above

`references/audit-prompt.md` is the auditors' single source of instructions — do not paraphrase it
into the spawn prompt, or the two will drift and the auditors will disagree with each other.

Auditors are **strictly read-only**. They surface verdicts with evidence and a proposed landing
form; they never call `AskUserQuestion` and never write. Approval and every write happen in this
integrator, single-threaded, so parallel auditors cannot contend.

If subagents cannot be spawned in the current environment, follow `references/audit-prompt.md`
inline per package instead; the rest of the flow is unchanged.

## Step 3 — Aggregate (read-only checkpoint)

Show the full surface before any decision, so the user sees the shape of the sweep rather than being
walked through unbounded one-by-one questions:

```text
comment-sweep 検出結果（scope: <X>, 対象 <n> ファイル / <m> パッケージ）

[<package>]  維持 <a> / 短縮 <b> / 削除 <c> / 移設 <d>
  ...（各 finding: 場所・対象コメント・判定・根拠・着地形）

移設先の内訳: docs/adr/ <p> 件 / docs/design/ <q> 件 / パッケージ README <r> 件
総 finding: <sum>（うち要判断 <k>）。これから 1 件ずつ確認します。
```

`維持` findings are reported as a count only — they need no decision, and listing them in full buries
the ones that do. If nothing needs action, say so plainly and stop.

## Step 4 — Per-item approval, then write (integrator-side)

For each non-`維持` finding, in descending impact order:

1. Present the finding with its evidence and the concrete landing form — show the comment **before**
   and **after** as a diff, and for a 移設 also show the exact prose that will be added to the
   destination document. A verdict the user cannot see the result of is not reviewable.
2. `AskUserQuestion` with the options the auditor surfaced (typically 移設 / 短縮 / 削除 / 維持 / 判断を保留).
3. On approval, write in this order — **destination document first, code second**. Reversing it
   creates a window where the rationale exists nowhere, and if the run is interrupted there, the
   reasoning is simply gone. When the auditor found the destination **already states the content**
   (`追記なし`), there is no document write: the finding lands as a 短縮 to the residue plus a link,
   and only the code changes.
4. If the destination is an **ADR**, do not decide the ADR's shape yourself. `docs/adr/README.md`
   declares accepted ADRs immutable, so a new rationale can land either as a new record or as a
   rewrite of an existing one, and which is right is a repository-policy call, not a code-cleanup
   call. Ask: 「既存 ADR-NNNN を書き換える」 / 「新規 ADR を起こす」 / 「ADR ではなく docs/design か README へ」 / 「今回は移設しない」.
   Whichever is chosen, the English canonical file and its `.ja.md` translation — plus the log table in
   `docs/adr/README.md` and `docs/adr/README.ja.md` — are updated in the same change.
5. If the destination is a package README, and the addition materially changes what that README
   claims, mention that `back-prop` is the right follow-up to check the README against code reality.

Never batch-apply without per-item confirmation. The judgments here are close calls by construction —
the obvious ones were already handled by `comment-reviewer` at diff time.

## Step 5 — Verify

- `make fix` then `make lint` over the touched packages — `revive exported` will catch a doc comment
  deleted where the convention requires one.
- `make md-lint` when a Markdown destination was written.
- Re-read each edited comment once: does the residue still stand on its own for someone who does not
  follow the link? A residue that only makes sense after reading the ADR has been cut too far, and
  that failure is invisible to every linter.

## Explicitly out of scope

- **`// Name は、〜です。` field comments** — `// Limit は、取得件数の上限です。`, `// StatusCode は、HTTP ステータスコードです。`.
  These are name restatements and carry little information, but the repo deliberately keeps one line
  per field for visual uniformity, and that call has been made. The bloat this skill exists to fix
  lives in long-form Why, not here. Do not flag them, do not propose deleting them, and do not
  reopen the question as a side note — it has already been decided.
- **Package overviews** — a `// Package …` comment, wherever it lives. Usage and How belong there and
  `docs/rules.md` exempts them. Note that this repository has **no `doc.go`**: all 300-plus overviews
  sit at the top of an ordinary source file, so an exclusion phrased as a filename excludes nothing.
- **Generated files** (`**/*.gen.go`, `*.sql.go`, `*_mock.go`) and `*_test.go`.
- **Functional / directive comments** — `//go:generate`, `//nolint`, `//go:build`, `//go:embed`,
  `// Code generated ... DO NOT EDIT`, shebangs, tool directives. These are not prose.
- **Docs prose quality** — `doc-reviewer` owns it. This skill only *adds* to a destination document;
  it does not audit what is already there.

## Relationship to the existing reviewers

| | Scope | Verdicts | Owns |
| --- | --- | --- | --- |
| `comment-reviewer` (via `impl-review`) | diff | 削除 / 書換 / 加筆 | generation-time gate; keeps new noise out |
| `doc-reviewer` | `README*` / `docs/**` | content findings | quality of docs prose |
| **`comment-sweep`** (this skill) | **stock** | **維持 / 短縮 / 削除 / 移設** | **jurisdiction — where content belongs** |

`comment-reviewer` stays as-is; it is the only thing stopping the inflow, and it is correctly scoped
to diffs. This skill is the outflow that never existed.

All user-visible output — findings, questions, proposed prose, summaries — is written in **Japanese**
per `CLAUDE.md`.
