---
name: new-spec-domain
description: Create the domain layer spec template at `docs/spec/<feature>/domain.md`. Asks for feature name (kebab-case) + aggregate name (PascalCase) via `ask the user explicitly`, reads the section structure for the domain layer from `.codex/scaffold-spec/domain-spec.md`, then writes a Markdown file with YAML code-block placeholders and TODO markers for the user to fill in (Overview / Entity / Cross-field Invariants / Behavior Methods / Value Objects / Repository Methods). NEVER overwrites an existing file. Does not invent business content — gathers identity only. Reads the spec format file at runtime so format changes propagate automatically.
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

- `.codex/scaffold-spec/domain-spec.md` — canonical section list for the domain layer.
- `docs/spec/<feature>/` — checks whether `domain.md` already exists.

**Writes (with confirmation)**:

- `docs/spec/<feature>/domain.md` — template file.

**Never touches**:

- Existing `domain.md` (aborts if found).
- Any other layer's spec file.

## First Step: Confirm Identity

This skill **MUST call `ask the user explicitly` immediately after invocation** (unless invoked from `new-spec` integrator with the feature name in context):

1. **Feature name** — free-text, kebab-case. Validate `^[a-z][a-z0-9-]*$`.
2. **Aggregate name** — free-text, PascalCase (e.g., `User`, `Order`). Becomes the Go struct name.

If `docs/spec/<feature>/domain.md` exists → abort. Otherwise create `docs/spec/<feature>/` if missing.

## Step 1. Read Section Definitions

Read `.codex/scaffold-spec/domain-spec.md` and extract the canonical section list. NEVER hardcode — this file is the source of truth.

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

Use the example output shown in `.codex/scaffold-spec/domain-spec.md` as the literal template format.

## Step 3. Confirm and Write

Display proposed path + first ~20 lines of template, then ask:

- 「以下の内容で `docs/spec/<feature>/domain.md` を作成しますか？」
- Options: 「作成する」 / 「キャンセル」

If approved, `mkdir -p docs/spec/<feature>` and `Write` the file.

## Step 4. Closing

```text
docs/spec/<feature>/domain.md を作成しました。次は editor で TODO を埋めてください。
usecase spec も必要なら new-spec-usecase または統合 new-spec を使ってください
（lean A 構成: controller / infra spec は不要、OpenAPI gen + sqlc gen から導出されます）。
```

## AI Modification Scope

- Write scope: new files under `docs/spec/<feature>/` only.
- Aborts if `domain.md` exists.

## Constraints

- ❌ Overwrite an existing `domain.md`
- ❌ Invent business content (fields / methods)
- ❌ Hardcode section list (always read `.codex/scaffold-spec/domain-spec.md`)
- ❌ Skip the identity `ask the user explicitly`
- ❌ Touch any layer other than `domain.md`
- ✅ Japanese user-facing output
- ✅ Validate feature name kebab-case + aggregate name PascalCase

## Checklist

- [ ] Feature name + aggregate name confirmed via `ask the user explicitly`
- [ ] `.codex/scaffold-spec/domain-spec.md` read for current section list
- [ ] `domain.md` did NOT already exist (skill aborts otherwise)
- [ ] Template written with H2 sections + YAML codeblocks + TODO markers
- [ ] Final summary in Japanese
- [ ] Only `docs/spec/<feature>/domain.md` was written
