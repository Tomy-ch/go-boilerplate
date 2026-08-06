---
name: glossary
description: >-
  Draft and maintain the business-vocabulary spec at `docs/spec/glossary.md` — the cross-cutting Ubiquitous Language page that per-feature specs cannot be, because one word meaning two things and two words meaning one only happen across features. Extracts a term inventory deterministically from the YAML in `docs/spec/*/domain.md`, the exported types AND the exported behaviours under `internal/domain/**` (accessors subtracted, so the verbs a noun-only sweep would lose survive), and the published names in `openapi/`, then reports four findings a machine can settle — terms with no row, orphan code symbols the specs never mention, declared code symbols that no longer resolve (both the specs' `package` / `struct` and the glossary's own code-symbol column, which nothing else verifies), and one identifier defined in two features — and asks a human to decide each. Use it when adding a feature and its new terms need registering, when someone asks what a business word means or whether two words are the same thing, when the glossary looks stale against the code, or for a periodic vocabulary sweep. Japanese triggers apply too — 「用語集を作って」「新出用語を登録」「この語の定義は」「orphan を出して」「語彙の棚卸し」. It never chooses the canonical name and never declares two words synonymous: identifier collision is mechanical, but whether two definitions actually differ is a reading and which name wins is a decision. Do NOT use it to write a feature's own spec (`new-spec` / `new-spec-domain`), to validate spec format (`verify-spec`), to audit DDD patterns against Evans (`ddd-audit`), or to detect business vocabulary that leaked into READMEs and ADRs (that is `back-prop`'s glossary detector).
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

Four sources, each read at runtime:

```sh
# terms declared by the specs
grep -n '^package:\|^struct:' docs/spec/*/domain.md

# exported types in the domain — the nouns
grep -rn '^type [A-Z][A-Za-z0-9]* \(struct\|interface\)' internal/domain --include='*.go'

# exported behaviours in the domain — the verbs
grep -rn '^func \(([a-z]* \*\?[A-Z][A-Za-z0-9]*) \)\?[A-Z][A-Za-z0-9]*(' internal/domain --include='*.go'

# published names
ls openapi/components/schemas/
```

Excluding `_test.go` and `mock/`. Read `.claude/scaffold-spec/domain-spec.md` at runtime for the
YAML shape rather than assuming it — the section list is that file's to change.

**Subtract the accessors from the behaviour set.** A method named after a field hands that field
back and says nothing its own name did not already say — `Name()`, `Email()`, `PublishedAt()`.
Compare each method name against the receiver type's field names and drop the matches. What survives
carries a verb or a judgement: `Cancel`, `IsCanceled`, `IsLowStock`, `UpdateEmail`.

Take the verbs as seriously as the nouns; they are the easier half to lose. **A vocabulary of nouns
can name what the business has and cannot say what happens to it** — and the rules live in what
happens. An inventory drawn from types alone reads as complete while every rule is missing from it.

Do not cut the constructors out mechanically. `New` is Go's word for construction, and when the
business calls the same act something else, that mismatch is a finding worth putting in front of a
person rather than a row to suppress. They will dominate a first sweep; the Mechanism vocabulary
section is where the ones that are genuinely construction go, and Step 1 subtracts them from then on.

A term's owner is the feature directory the spec lives in plus the aggregate it declares. Two owners
for one term is not a data problem to reconcile; it is the finding.

## Step 3. Four findings, kept apart

They ask for different things, so do not merge them into one list.

- **新出用語** — declared by a spec, no row in the glossary. Propose a row with the definition drafted
  from the spec's own prose, and let the human rewrite it. The definition is the part that must not
  read like generated text.
- **orphan** — exported in `internal/domain/**` — a type, or a behaviour that survived the accessor
  cut — absent from every spec and from the glossary, and not listed as mechanism vocabulary. This is
  the one finding that catches what nobody wrote down: everything else starts from a document, so a
  term that was never documented is invisible to it. Each orphan resolves one of three ways — it
  becomes a row, it goes to Mechanism vocabulary, or it is a naming mistake in the code.
- **解決しない参照** — a declared code symbol that no longer resolves. Two sources, checked the same
  way: a spec's `package` / `struct`, and **the glossary's own code-symbol column**. That column
  takes four shapes and all four must resolve — `package.Type`, `package.Type.Method`,
  `package.Func`, and `package.Value` for a package-level constant or variable, which is how a
  named state or role is usually carried. A checker that knows only the first three reports a
  false defect against the rows that matter most. Deterministic — settle it with `grep`, never with a
  judgement call. The glossary side is the one that matters most and the one nobody was checking:
  the page declares that it governs the code, and **a governing claim never compared against the
  thing it governs is decoration.**
- **同音異義** — one identifier declared by two features. Report both definitions side by side and
  ask whether they are actually the same concept. Never answer that question here.

## Step 4. Decide per finding

`AskUserQuestion`, batched at most 4 findings per call. Options follow what the finding supports —
for an orphan: 「用語として登録」/「機構語として記録」/「コード側の命名が誤り」/「今回は保留」.

For a new term, present the drafted definition and make rewriting it the easy path. **A definition
nobody edited is a definition nobody agreed to.**

For an unresolved code symbol the options are ordered, and the order is the whole point:

1. 「コード側の改名が誤り。シンボルを表に合わせて戻す」
2. 「業務側で語の扱いが変わった。**先に表の行を改訂し**、そのあとコードを追随させる」
3. 「行が実体を失った（機能ごと消えた）ので行を削除する」
4. 「今回は保留」

**Never offer "rewrite the row to match the code" as an option of its own.** It is the cheapest fix
on the table at the exact moment the page is most vulnerable, and taking it turns the glossary into
an index of the code. An index cannot contradict what it indexes, and a vocabulary that cannot
contradict the code can never tell anyone the model is wrong. Option 2 reaches the same end state
through the order the page requires — the language moves first and the code follows.

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
- ❌ 在庫を型だけから取る（動詞が落ち、業務規則が宿る側が丸ごと見えなくなる）
- ❌ 用語表のコードシンボル列を検証せずに済ませる（照合されない統べる主張は装飾）
- ❌ 「表をコードに合わせて書き換える」を単独の選択肢として出す（索引への退化はこの瞬間に開く）
- ❌ 器が無いときに器を作る（規則を含む設計行為であり、このスキルの担当ではない）
- ❌ `docs/spec/glossary.md` 以外への書き込み、`.ja.md` ペアの作成、commit / push
- ✅ 4 種の findings を分けて報告（求められることが違うため）
- ✅ サンプル由来の行は `sample-api` マーカーの内側へ
- ✅ 出力は日本語

## Checklist

- [ ] スコープを `AskUserQuestion` で確認
- [ ] 器を読み、既存行・Mechanism vocabulary・Watch list を取り出す（無ければ停止）
- [ ] spec YAML / domain 公開型 / domain 公開振る舞い / OpenAPI から実行時に抽出
- [ ] 振る舞いからアクセサ（フィールド名と同名）を差し引く
- [ ] 用語表のコードシンボル列を grep で解決確認
- [ ] 4 種の findings を分け、orphan から Mechanism vocabulary を差し引く
- [ ] findings ごとに人が判断（定義文は書き換えられる形で提示）
- [ ] `docs/spec/glossary.md` のみ更新、サンプル行はマーカー内側
- [ ] 日本語で報告し、他スキルの担当は名指しして終わる
