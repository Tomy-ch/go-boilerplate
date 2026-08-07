---
name: new-spec-domain
description: Create the domain layer spec template at `docs/spec/<feature>/domain.md`. Asks for feature name (kebab-case) + aggregate name (PascalCase) via `AskUserQuestion`, reads the section structure for the domain layer from `.claude/scaffold-spec/domain-spec.md`, then writes a Markdown file with YAML code-block placeholders and TODO markers for the user to fill in (Overview / Entity / Cross-field Invariants / Behavior Methods / Value Objects / Repository Methods). NEVER overwrites an existing file. Does not invent business content — gathers identity only. Reads the spec format file at runtime so format changes propagate automatically. After writing the template it also registers the aggregate in the business-vocabulary spec `docs/spec/glossary.md` — one row and no more, since every other term is still a TODO — reporting a homonym (the same word already owned by another feature) back to the user instead of resolving it, because which name wins is a decision about how the business will talk and not something a generator may settle. `new-spec-usecase` deliberately has no equivalent step: a usecase spec declares Interfaces, DTOs and Workflows, which are application-layer names rather than words of the business.
---

# New Spec — Domain

Create the domain layer spec template for one feature.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Starting a new feature and need the domain layer spec template.
- Adding the domain spec to an existing feature directory (e.g., other layer specs are already in place).

Do NOT use this skill for:

- Editing an existing `domain.md` — open in the editor directly.
- Generating Go code from spec — that's `scaffold-domain`.
- Validating spec consistency — that's `verify-spec`.
- Creating both layer specs (domain + usecase) in one go — use the integrator `new-spec`. lean A only requires these two specs; controller / infra are derived from OpenAPI gen + sqlc gen, no spec file.

## What This Skill Reads / Writes

**Reads (always)**:

- `.claude/scaffold-spec/domain-spec.md` — canonical section list for the domain layer.
- `docs/spec/<feature>/` — checks whether `domain.md` already exists.
- `docs/spec/glossary.md` — the business-vocabulary spec, to check the aggregate name against.

**Writes (with confirmation)**:

- `docs/spec/<feature>/domain.md` — template file.
- `docs/spec/glossary.md` — one row for the aggregate, only when the user approves it.

**Never touches**:

- Existing `domain.md` (aborts if found).
- Any other layer's spec file.
- Any glossary row other than the one being added.

## First Step: Confirm Identity

This skill **MUST call `AskUserQuestion` immediately after invocation** (unless invoked from `new-spec` integrator with the feature name in context):

1. **Feature name** — free-text, kebab-case. Validate `^[a-z][a-z0-9-]*$`.
2. **Aggregate name** — free-text, PascalCase (e.g., `User`, `Order`). Becomes the Go struct name.

If `docs/spec/<feature>/domain.md` exists → abort. Otherwise create `docs/spec/<feature>/` if missing.

## Step 1. Read Section Definitions

Read `.claude/scaffold-spec/domain-spec.md` and extract the canonical section list. NEVER hardcode — this file is the source of truth.

Current section list (mirror dynamically):

1. Overview
2. Entity
3. Cross-field Invariants
4. Behavior Methods
5. Value Objects
6. Repository Methods

## Step 2. Generate the Template

Assemble Markdown with H1 title `<FeatureName Display> — Domain Spec`, then per section:

- Overview: single `TODO:` line
- Entity: YAML code block with `package` / `struct` / `fields` (1 placeholder field)
- Cross-field Invariants: bullet list with `TODO:` line
- Behavior Methods: YAML code block with placeholder method
- Value Objects: YAML code block with note "利用しない場合は節ごと削除"
- Repository Methods: YAML code block with placeholder method

Use the example output shown in `.claude/scaffold-spec/domain-spec.md` as the literal template format.

## Step 3. Confirm and Write

Display proposed path + first ~20 lines of template, then ask:

- 「以下の内容で `docs/spec/<feature>/domain.md` を作成しますか？」
- Options: 「作成する」 / 「キャンセル」

If approved, `mkdir -p docs/spec/<feature>` and `Write` the file.

## Step 3.5. Register the aggregate in the glossary

**Only the aggregate name is a term at this point.** This skill writes a template of TODOs; the
fields, value objects and behaviours are not decided yet, so there is nothing else to register. One
row, or none.

Read `docs/spec/glossary.md` and compare the aggregate name against the existing rows.

- **Already present with a different owner** — a homonym. Report both rows side by side and stop:
  the same word owned by two features is the finding this page exists to surface, and which name
  wins is a decision about how the business will talk. **Never resolve it here.**
- **Already present with the same owner** — nothing to do; say so.
- **Not present** — propose one row, then confirm with `AskUserQuestion`
  （「用語表へ追加する」/「今回は追加しない」）.

Draft the definition from the feature and aggregate names and **say plainly that it is a draft**. A
definition nobody edited is a definition nobody agreed to; the row is worth more empty than
plausibly wrong.

Place the row inside the `sample-api:begin` / `sample-api:end` markers when the feature is
sample-derived, outside them otherwise. A row on the wrong side of a marker either vanishes with the
sample or outlives it.

## Step 4. Closing

```text
docs/spec/<feature>/domain.md を作成しました。次は editor で TODO を埋めてください。
用語表には集約名だけを登録しました。TODO を埋めると値オブジェクトや状態の語が現れるので、
そのときは /glossary で用語表へ反映してください（新出用語・orphan・同音異義を出します）。
usecase spec も必要なら new-spec-usecase または統合 new-spec を使ってください
（lean A 構成: controller / infra spec は不要、OpenAPI gen + sqlc gen から導出されます）。
```

**`new-spec-usecase` has no equivalent step, deliberately.** A usecase spec declares an Interface,
DTOs and a Workflow — application-layer names like `CreatePurchase` or `PurchaseView`, which do not
pass the glossary's bar (would someone who knows the business recognise it as a word of the
business?). The domain layer introduces terms; the usecase layer uses them.

That holds **while a domain layer exists to do the introducing.** A projection-only feature — no
aggregate, no `domain.md`, a QueryService and nothing else — has no such layer, and its words
(sales, ranking, a postal-code lookup) would otherwise be introduced by nobody. `/glossary` covers
that case from the read side. It is not this skill's, because this skill runs precisely when an
aggregate is being created.

## AI Modification Scope

- Write scope: new files under `docs/spec/<feature>/`, plus one appended row in
  `docs/spec/glossary.md`.
- Aborts if `domain.md` exists.

## Constraints

- ❌ Overwrite an existing `domain.md`
- ❌ Invent business content (fields / methods)
- ❌ Hardcode section list (always read `.claude/scaffold-spec/domain-spec.md`)
- ❌ Skip the identity `AskUserQuestion`
- ❌ Touch any layer other than `domain.md`
- ❌ Resolve a glossary homonym, or decide which name wins
- ❌ Register anything beyond the aggregate name (the rest are still TODOs)
- ❌ Write a definition without saying it is a draft to be rewritten
- ✅ Japanese user-facing output
- ✅ Validate feature name kebab-case + aggregate name PascalCase

## Checklist

- [ ] Feature name + aggregate name confirmed via `AskUserQuestion`
- [ ] `.claude/scaffold-spec/domain-spec.md` read for current section list
- [ ] `domain.md` did NOT already exist (skill aborts otherwise)
- [ ] Template written with H2 sections + YAML codeblocks + TODO markers
- [ ] Aggregate name checked against `docs/spec/glossary.md` (homonym → report and stop)
- [ ] Glossary row added only after `AskUserQuestion`, on the correct side of the sample markers
- [ ] Final summary in Japanese, handing the remaining terms to `/glossary`
- [ ] Only `docs/spec/<feature>/domain.md` and one glossary row were written
