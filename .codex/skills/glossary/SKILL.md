---
name: glossary
description: >-
  Draft and maintain `docs/spec/glossary.md`, the cross-feature Ubiquitous Language spec for business vocabulary. Use when a feature introduces terms that need registering, someone asks what a business term means or whether terms conflict, the glossary may be stale against specs/code/OpenAPI, or a periodic vocabulary sweep is needed; Japanese triggers include 「用語集を作って」「新出用語を登録」「この語の定義は」「orphan を出して」「語彙の棚卸し」. At runtime, inventory spec declarations, exported domain types and behaviours (with accessors subtracted), and OpenAPI schemas; separately verify unresolved code symbols declared in both feature specs and the glossary itself, and report new terms, undocumented code orphans, unresolved references, and homonyms for human decisions. Never choose a canonical name or declare synonyms. Do NOT use to author a feature spec (`new-spec` / `new-spec-domain`), validate spec format (`verify-spec`), audit Evans DDD patterns (`ddd-audit`), or find glossary drift in READMEs and ADRs (`back-prop`).
---

# Glossary

Maintain `docs/spec/glossary.md` as the single source of truth for this system's business vocabulary.

## Decision boundary

Settle only mechanical facts: which identifiers exist, lack a glossary row, appear under two owners, or no longer resolve. **Never choose the canonical name and never declare two words synonymous.**

This is an evidence boundary, not caution. Identifier collision is a string comparison; deciding whether two definitions differ requires reading prose. Different words for one concept leave no mechanical trace. Choosing a winning name decides how the business speaks, and code cannot supply that decision. Report the evidence and proposed wording, then leave the decision to a person.

## Scope and baseline

1. Accept `--feature <name>` for a single feature; otherwise cover every feature. In an interactive run, use Codex's native user-input interaction to choose between these scopes. Keep each interaction to at most four findings. When no interaction is available, report rather than decide any finding left open.
2. Read `docs/spec/glossary.md` before extracting anything. Derive, rather than hardcode, its existing rows, **Mechanism vocabulary**, and **Watch list**.
3. If the container is absent, report that and stop. Creating this container is a design act with its own rules; this skill fills it and does not invent it.

The Mechanism vocabulary is the suppression channel. **Subtract its names from every orphan set before reporting.** Otherwise every sweep repeats structural names and becomes unreadable.

## Extract the runtime inventory

Read `.claude/scaffold-spec/domain-spec.md` at runtime to learn the domain-spec YAML shape; do not assume its sections stay fixed. Extract four inventories:

- `package:` and `struct:` declarations from the YAML in `docs/spec/*/domain.md`, restricted to the selected feature when applicable;
- exported `type X struct` and `type X interface` declarations in `internal/domain/**`, excluding `_test.go` and `mock/`;
- exported behaviours in `internal/domain/**`, excluding `_test.go` and `mock/`:

  ```sh
  grep -rn '^func \(([a-z]* \*\?[A-Z][A-Za-z0-9]*) \)\?[A-Z][A-Za-z0-9]*(' internal/domain --include='*.go'
  ```

  Subtract accessors: compare each method name to the fields of its receiver type and drop exact
  matches such as `Name()`, `Email()`, and `PublishedAt()`. The remainder expresses a verb or a
  judgement, for example `Cancel`, `IsCanceled`, `IsLowStock`, or `UpdateEmail`.
- published names from `openapi/components/schemas/`.

Do not mechanically discard constructors. `New` is Go's word for construction; if the business calls
the same action something else, that mismatch is a finding. Genuine construction mechanisms belong
in Mechanism vocabulary and Step 1 will suppress them in subsequent runs.

Treat verbs as seriously as nouns. A noun-only vocabulary says what the business has but not what
happens to it; rules live in what happens. An inventory drawn only from types can appear complete
while missing that entire side of the business.

Treat a term's owner as the feature directory plus its declared aggregate. Do not reconcile two owners: that condition is a finding.

## Keep the four findings separate

Do not merge these lists: each asks the human for a different decision.

- **新出用語** — a spec declares a term with no glossary row. Draft a candidate row from the spec's own prose and make editing the definition easy. A definition nobody edited is a definition nobody agreed to.
- **orphan** — an exported domain type, or a behaviour that survived the accessor subtraction, appears in no spec, no glossary row, and no Mechanism vocabulary entry. This alone catches vocabulary that nobody documented. Classify each as a glossary row, Mechanism vocabulary, or a code naming mistake; do not make the code change.
- **解決しない参照** — a declared code symbol no longer resolves. Verify both a feature spec's `package` / `struct` and the glossary's own code-symbol column with `grep`. That column takes four shapes and all four must resolve — `package.Type`, `package.Type.Method`, `package.Func`, and `package.Value` for a package-level constant or variable, which is how a named state or role is usually carried; a checker that knows only the first three reports a false defect against the rows that matter most, never a judgement call. This is always a defect, but do not decide whether the glossary, spec, or code is wrong. The glossary is the governing claim; a governing claim never compared with what it governs is decoration.
- **同音異義** — one identifier is declared by two features. Put both definitions and owners side by side and ask whether they are the same concept. Never answer that question here.

For each interactive decision, show only the evidence required and batch no more than four findings. For new terms, preserve a directly editable proposed definition. For an orphan, offer: register it as a term, record it as Mechanism vocabulary, flag code naming, or leave it pending.

For an unresolved code symbol, offer these options in this order:

1. 「コード側の改名が誤り。シンボルを表に合わせて戻す」
2. 「業務側で語の扱いが変わった。**先に表の行を改訂し**、そのあとコードを追随させる」
3. 「行が実体を失った（機能ごと消えた）ので行を削除する」
4. 「今回は保留」

Never offer “rewrite the row to match the code” as an option of its own. That turns the glossary
into an index of code: an index cannot contradict what it indexes, so it cannot reveal that the
model is wrong. The second option preserves the required order: revise the language first, then
make the code follow it.

## Write and close

Write **only** `docs/spec/glossary.md`; do not edit feature specs, READMEs, ADRs, the DDD ledger, source, or generated files. Do not create a `.ja.md` pair for the glossary: this spec tree uses one Japanese file with English headings.

Put sample-derived rows between `sample-api:begin` and `sample-api:end`. Put terms that survive sample removal outside those markers.

Close in Japanese with the rows added, orphan classifications, unresolved references, homonyms left open, and follow-ups owned by other skills. Do not commit or push.

## Constraints

- ❌ Inventory only types (verbs disappear, along with the side where business rules live).
- ❌ Skip verification of the glossary's code-symbol column (a governing claim that is never compared is decoration).
- ❌ Offer “rewrite the row to match the code” as an option by itself (that moment opens the regression into an index).

## Checklist

- [ ] Choose all features or `--feature <name>`.
- [ ] Read the existing rows, Mechanism vocabulary, and Watch list; stop if the container is absent.
- [ ] Extract spec YAML, exported domain types, exported domain behaviours, and OpenAPI schemas at runtime.
- [ ] Subtract accessors from behaviours by matching method names to receiver field names.
- [ ] Verify the glossary code-symbol column with `grep`, along with spec `package` / `struct` declarations.
- [ ] Subtract Mechanism vocabulary from orphans.
- [ ] Report the four finding kinds independently and leave naming/synonym decisions to a person.
- [ ] Update only `docs/spec/glossary.md`; keep sample rows inside their markers.
- [ ] Finish in Japanese without a commit or push.
