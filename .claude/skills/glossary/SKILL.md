---
name: glossary
description: >-
  Draft and maintain the business-vocabulary spec at `docs/spec/glossary.md` — the cross-cutting Ubiquitous Language page that per-feature specs cannot be, because one word meaning two things and two words meaning one only happen across features. Extracts a term inventory deterministically from the YAML in `docs/spec/*/domain.md`, the exported types under `internal/domain/**`, and the published names in `openapi/`, then reports four findings a machine can settle — terms with no row, orphan code symbols the specs never mention, spec `package` / `struct` values that no longer resolve, and one identifier defined in two features — and asks a human to decide each. Use it when adding a feature and its new terms need registering, when someone asks what a business word means or whether two words are the same thing, when the glossary looks stale against the code, or for a periodic vocabulary sweep. Japanese triggers apply too — 「用語集を作って」「新出用語を登録」「この語の定義は」「orphan を出して」「語彙の棚卸し」. It never chooses the canonical name and never declares two words synonymous: identifier collision is mechanical, but whether two definitions actually differ is a reading and which name wins is a decision. Do NOT use it to write a feature's own spec (`new-spec` / `new-spec-domain`), to validate spec format (`verify-spec`), to audit DDD patterns against Evans (`ddd-audit`), or to detect business vocabulary that leaked into READMEs and ADRs (that is `back-prop`'s glossary detector).
argument-hint: '[--feature <name>]'
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, AskUserQuestion
---

# Glossary

Maintains `docs/spec/glossary.md`, the single source of truth for this system's business vocabulary.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory
(not loaded as a skill; for human reference only).

## What this skill decides and what it does not

It settles what a machine can settle: which identifiers exist, which are missing a row, which are
defined twice, which no longer resolve. **It never chooses the canonical name and never declares two
words synonymous.**

The split is not caution, it is where the evidence stops. Identifier collision is a string
comparison; whether the two definitions actually *differ* is a reading of prose. Two different words
naming one concept leaves no mechanical trace at all — it surfaces only when a person reads both rows
and recognises them. And which name wins is a decision about how the business will talk, which no
amount of code can supply.

Report the finding, propose wording, and hand the decision over.

## When to Use

- A feature landed and its new terms are not in the glossary.
- Someone asks what a business word means, or whether two words mean the same thing.
- The glossary looks stale against the code.
- Periodic vocabulary sweep.

Do NOT use for:

- A feature's own spec — `new-spec` / `new-spec-domain`; spec format — `verify-spec`.
- DDD patterns against Evans — `ddd-audit`.
- Business vocabulary that leaked into READMEs / ADRs — `back-prop`'s glossary detector.

## Step 0. Confirm scope

`AskUserQuestion`, one question:

```text
質問: どこまで見ますか？
選択肢:
  - 全体（既定。全 feature と domain 全体を突き合わせる）
  - 特定 feature のみ（--feature <name>。新出用語の登録向け）
```

## Step 1. Read the glossary as the baseline (deterministic — do not delegate)

Read `docs/spec/glossary.md`. It gives three things the run depends on, and none of them may be
hardcoded here:

- the existing rows (term, owner, code symbol, public name),
- the **Mechanism vocabulary** section — names already judged *not* to be business terms,
- the **Watch list** — homonyms and synonyms already known.

The Mechanism section is the suppression channel. **Subtract it from every orphan set before
reporting.** A sweep that re-proposes the same structural names every run is a sweep nobody reads,
which is the failure this section exists to prevent.

If the glossary does not exist, say so and stop. Creating the container is a design act with rules in
it; this skill fills a container, it does not invent one.

## Step 2. Extract the inventory

Three sources, each read at runtime:

```sh
# terms declared by the specs
grep -n '^package:\|^struct:' docs/spec/*/domain.md

# exported types in the domain
grep -rn '^type [A-Z][A-Za-z0-9]* \(struct\|interface\)' internal/domain --include='*.go'

# published names
ls openapi/components/schemas/
```

Excluding `_test.go` and `mock/`. Read `.claude/scaffold-spec/domain-spec.md` at runtime for the
YAML shape rather than assuming it — the section list is that file's to change.

A term's owner is the feature directory the spec lives in plus the aggregate it declares. Two owners
for one term is not a data problem to reconcile; it is the finding.

## Step 3. Four findings, kept apart

They ask for different things, so do not merge them into one list.

- **新出用語** — declared by a spec, no row in the glossary. Propose a row with the definition drafted
  from the spec's own prose, and let the human rewrite it. The definition is the part that must not
  read like generated text.
- **orphan** — exported in `internal/domain/**`, absent from every spec and from the glossary, and not
  listed as mechanism vocabulary. This is the one finding that catches what nobody wrote down:
  everything else starts from a document, so a term that was never documented is invisible to it.
  Each orphan resolves one of three ways — it becomes a row, it goes to Mechanism vocabulary, or it
  is a naming mistake in the code.
- **解決しない参照** — a spec's `package` or `struct` no longer exists. Deterministic and always a
  defect, though which side is wrong is not this skill's call.
- **同音異義** — one identifier declared by two features. Report both definitions side by side and
  ask whether they are actually the same concept. Never answer that question here.

## Step 4. Decide per finding

`AskUserQuestion`, batched at most 4 findings per call. Options follow what the finding supports —
for an orphan: 「用語として登録」/「機構語として記録」/「コード側の命名が誤り」/「今回は保留」.

For a new term, present the drafted definition and make rewriting it the easy path. **A definition
nobody edited is a definition nobody agreed to.**

## Step 5. Write

Write `docs/spec/glossary.md` only. The spec tree is Japanese single-file with English headings and
has **no `.ja.md` pair** — do not create one, and do not chain `canonicalize-doc`.

Sample-derived rows live between the `sample-api:begin` / `sample-api:end` markers so they leave with
the sample; a row for a term that survives sample removal goes outside them. Getting this wrong is
how the page starts describing terms that no longer exist.

Do not touch feature specs, READMEs, ADRs, the DDD ledger, or source. A term that needs renaming in
code is a follow-up to surface, not an edit to make here.

## Step 6. Closing

Report in Japanese: rows added, orphans classified, unresolved references, homonyms left open. Name
the follow-ups that belong to other skills and stop there. No commit, no push.

## AI Modification Scope

- Read: `docs/spec/**`, `internal/domain/**`, `openapi/**`, `.claude/scaffold-spec/domain-spec.md`
- Write: `docs/spec/glossary.md` only
- Never touch: feature specs, READMEs, ADRs, the DDD ledger, source code, generated files, `AGENTS.md`

## Constraints

- ❌ 正名をスキルが決める（どの名前が勝つかは業務の話し方の決定）
- ❌ 同義を機械が判定する（機械的痕跡が無い。提示までにとどめる）
- ❌ Mechanism vocabulary を差し引かずに orphan を報告する（毎回同じ一覧が出て読まれなくなる）
- ❌ 器が無いときに器を作る（規則を含む設計行為であり、このスキルの担当ではない）
- ❌ `docs/spec/glossary.md` 以外への書き込み、`.ja.md` ペアの作成、commit / push
- ✅ 4 種の findings を分けて報告（求められることが違うため）
- ✅ サンプル由来の行は `sample-api` マーカーの内側へ
- ✅ 出力は日本語

## Checklist

- [ ] スコープを `AskUserQuestion` で確認
- [ ] 器を読み、既存行・Mechanism vocabulary・Watch list を取り出す（無ければ停止）
- [ ] spec YAML / domain 公開型 / OpenAPI から実行時に抽出
- [ ] 4 種の findings を分け、orphan から Mechanism vocabulary を差し引く
- [ ] findings ごとに人が判断（定義文は書き換えられる形で提示）
- [ ] `docs/spec/glossary.md` のみ更新、サンプル行はマーカー内側
- [ ] 日本語で報告し、他スキルの担当は名指しして終わる
