---
name: new-spec-usecase
description: Create the usecase layer spec template at `docs/spec/<feature>/usecase.md`. Asks for feature name (kebab-case) + usecase package name (lowercase, e.g., `user`) + Usecase interface name (default `Usecase`) via `ask the user explicitly`, reads the section structure for the usecase layer from `.codex/scaffold-spec/usecase-spec.md`, then writes a Markdown file with YAML code-block placeholders and TODO markers (Overview / Interface / DTOs / Dependencies / Workflow). NEVER overwrites an existing file. Does not invent business content — gathers identity only. Reads the spec format file at runtime so format changes propagate automatically.
---

# New Spec — Usecase

Create the usecase layer spec template for one feature.

A Japanese reference translation of this skill is available at `SKILL.ja.md` in the same directory (not loaded as a skill; for human reference only).

## When to Use

- Starting a new feature and need the usecase layer spec template.
- Adding the usecase spec to an existing feature directory.

Do NOT use this skill for:

- Editing an existing `usecase.md` — open in the editor directly.
- Generating Go code from spec — that's `scaffold-usecase`.
- Validating spec consistency — that's `verify-spec`.
- Creating both layer specs (domain + usecase) in one go — use the integrator `new-spec`. lean A only requires these two specs; controller / infra are derived from OpenAPI gen + sqlc gen, no spec file.

## What This Skill Reads / Writes

**Reads (always)**:

- `.codex/scaffold-spec/usecase-spec.md` — canonical section list for the usecase layer.
- `docs/spec/<feature>/` — checks whether `usecase.md` already exists.

**Writes (with confirmation)**:

- `docs/spec/<feature>/usecase.md` — template file.

**Never touches**:

- Existing `usecase.md` (aborts if found).
- Any other layer's spec file.

## First Step: Confirm Identity

This skill **MUST call `ask the user explicitly` immediately after invocation** (unless invoked from `new-spec` integrator with the feature name in context):

1. **Feature name** — free-text, kebab-case. Validate `^[a-z][a-z0-9-]*$`.
2. **Usecase package name** — free-text, lowercase (e.g., `user`, `order`).
3. **Usecase interface name** — free-text, default `Usecase` (PascalCase).

If `docs/spec/<feature>/usecase.md` exists → abort. Otherwise create `docs/spec/<feature>/` if missing.

## Step 1. Read Section Definitions

Read `.codex/scaffold-spec/usecase-spec.md` for the canonical section list:

1. Overview
2. Interface
3. DTOs
4. Dependencies
5. Workflow

NEVER hardcode — re-read the spec format file each run.

## Step 2. Generate the Template

Assemble Markdown with H1 title `<FeatureName Display> — Usecase Spec`, then per section:

- Overview: single `TODO:` line
- Interface: YAML code block with `package` / `name` / `methods` (1 placeholder method)
- DTOs: YAML code block with placeholder DTO struct
- Dependencies: YAML code block with placeholder boundary list
- Workflow: H3 per method with YAML block (`tx_required` / `steps` / `calls` / `errors`)

Use the example output shown in `.codex/scaffold-spec/usecase-spec.md` as the literal template format.

## Step 3. Confirm and Write

Display proposed path + first ~20 lines, then ask:

- 「以下の内容で `docs/spec/<feature>/usecase.md` を作成しますか？」
- Options: 「作成する」 / 「キャンセル」

If approved, `mkdir -p docs/spec/<feature>` and `Write` the file.

## Step 4. Closing

```text
docs/spec/<feature>/usecase.md を作成しました。次は editor で TODO を埋めてください。
domain spec も必要なら new-spec-domain または統合 new-spec を使ってください
（lean A 構成: controller / infra spec は不要、OpenAPI gen + sqlc gen から導出されます）。
```

## AI Modification Scope

- Write scope: new files under `docs/spec/<feature>/` only.
- Aborts if `usecase.md` exists.

## Constraints

- ❌ Overwrite an existing `usecase.md`
- ❌ Invent business content (methods / DTOs / workflow)
- ❌ Hardcode section list (always read `.codex/scaffold-spec/usecase-spec.md`)
- ❌ Skip the identity `ask the user explicitly`
- ❌ Touch any layer other than `usecase.md`
- ✅ Japanese user-facing output
- ✅ Validate feature name kebab-case + package name lowercase

## Checklist

- [ ] Feature + package + interface name confirmed via `ask the user explicitly`
- [ ] `.codex/scaffold-spec/usecase-spec.md` read for current section list
- [ ] `usecase.md` did NOT already exist (skill aborts otherwise)
- [ ] Template written with H2 sections + YAML codeblocks + TODO markers
- [ ] Final summary in Japanese
- [ ] Only `docs/spec/<feature>/usecase.md` was written
