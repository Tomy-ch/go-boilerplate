# Boilerplate-only Conventions

English | [日本語](../ja/get-started/boilerplate-only-conventions.ja.md)

This file collects the statements in this repository whose premise holds only while it *is* the
upstream boilerplate — that it is a template, that its readers are forks, that its sample feature
set exists to be harvested and then removed. **Setup deletes this file whole** (see
[setup-repository.md](setup-repository.md) Phase 16). Nothing written here is a rule for a project
built from the template.

What survives setup is the general form of each rule, stated in the document that owns it:
[`docs/adr/README.md`](../adr/README.md), [`docs/rules.md`](../rules.md), the layer READMEs. This
file records only the deviations the upstream takes from those general forms — because a deviation
stated in a surviving document becomes a lie the moment the repository is forked.

## Why the deviations are collected here rather than marked in place

Marking each deviation where it stands would have the removal script cutting regions out of prose
that was written around them. What is removed is a region; what breaks is the text on either side
of it, and every later edit near a marker is a fresh chance to break it unnoticed.

Collecting them into one file that is deleted whole has no such failure mode: the surviving
documents never contained the premise, so nothing needs repairing after the cut. Only the pointers
back to this file stay in place, and each is a single self-contained line carrying a
`boilerplate-only:line` marker, so removing one cannot disturb the text around it.

## Marker convention

`boilerplate-only` is the one namespace for everything that stops being true when this repository is
forked, and `make setup-remove-boilerplate-identity` is the one pass that resolves it (Phase 12 of
[setup-repository.md](setup-repository.md)).

| Marker | Placement | Effect |
| --- | --- | --- |
| `boilerplate-only:line` | trailing comment on the line it applies to | that line is removed |
| `boilerplate-only:begin` / `boilerplate-only:end` | own-line comments around a region | the region and both markers are removed |
| `boilerplate-only:replace-begin` / `replace-with` / `replace-end` | own-line comments around two regions | the first region is removed and the second, written as commented-out lines, is uncommented in its place |

Reach for `replace-*` when deleting the region would take a heading or a rule down with it — where
the fork needs *something* said, not nothing. In Markdown the commented-out lines take the form
`<!-- = ... -->`; `# = ...` renders as a second top-level heading and fails markdownlint MD025, and
`// = ...` renders as literal text.

Comment form follows the existing `sample-api` markers — `<!-- ... -->` in Markdown, `//` or `#` in
code — so the same scanner shape works. The two namespaces are **not** interchangeable and must not
be stripped by one pass: they fire at different moments (`boilerplate-only` when the template is set
up, `sample-api` when the sample feature set is removed), and a fork may reasonably do one without
the other.

The removal **scans the repository** rather than working from a list of files. A list is something a
marker can be written outside of, and the failure is silent: the pass reports success, and the
premise reaches the fork with nothing to announce it. The only files exempt are dependency
checkouts and generated output, declared in
[`scripts/setup/remove-boilerplate-identity/targets.ts`](../../scripts/setup/remove-boilerplate-identity/targets.ts).

## The premise

> This is a **template** repository: downstream users fork it and need to understand *why* each
> choice was made, and to *supersede* individual choices with their own without editing a shared
> monolith.

Every deviation below rests on that sentence, and that sentence is false for the repository you are
setting up.

## ADR conventions that apply only upstream

The general regime — one immutable record per file, replaced by adding a *new* ADR rather than by
editing the old one — is stated in [`docs/adr/README.md`](../adr/README.md) and is what a fork
inherits. While this repository is distributed as a boilerplate, it deviates as follows.

### Amendment in place

An `accepted` ADR is amended **in place**: update `date`, keep `status: accepted`, and do not create
a superseding ADR for what is still the same decision. What this repository ships is the current
design, not the sequence of positions that produced it, and a fork that must read three ADRs to
learn one rule pays for history it did not live. When an amendment changes the conclusion, the
position it replaces moves to Alternatives Considered with the reason it was dropped — nothing is
discarded, it changes section.

**This deviation does not transfer.** In a project built from the template, an ADR records a
decision that project actually took, and amending it in place destroys precisely the history
[ADR-0000](../adr/0000-record-architecture-decisions.md) exists to keep.

### A new ADR versus a revision

A new ADR is for a decision that should be read independently of the one beside it, not for a
revision of one. `superseded` stays in the lifecycle for a decision genuinely replaced rather than
revised.

### Consolidation pass (authorised, one-off per harvest)

This repository's sample feature set is developed, harvested, and then removed. Implementing a
sample produces ADRs that are part architectural decision and part feature detail, and they
accumulate at the tail of the numbering in discovery order — which is exactly what the ordering
convention in [`docs/adr/README.md`](../adr/README.md) exists to prevent. A **consolidation pass may
therefore merge, rewrite, and retire such ADRs**, feeding the architectural residue back into the
ordered set and moving the feature content to `docs/spec/`.

The pass is bounded: it applies only to ADRs produced by sample development, it is performed as one
reviewed change, and every retired ADR's architectural content survives in the ADR that absorbed it
— nothing is discarded, only relocated. A number freed by a retirement is never re-assigned to a
different decision: after a consolidation the range is contiguous again, but no surviving ADR
inherits a retired ADR's old number.

Outside a consolidation pass, an ADR that is still the same decision is amended in place per the
convention above. What this exception adds is the authority to merge and retire files, which an
amendment does not have.

**This deviation does not transfer** unless a fork adopts a harvest cycle of its own, which is its
own decision to take and to record.

### Exclusion ADRs at setup

Exclusion ADRs carry a `setup-review` tag so the repository-setup flow can enumerate them, and a
fork may edit them directly at setup to establish its own baseline rather than superseding them. The
instruction belongs to the fork, so it lives in [setup-repository.md](setup-repository.md) Phase 13;
it is named here only because the tag is a boilerplate-side device that means nothing once setup is
done.

## What applies after setup

Once this file is gone, [`docs/adr/README.md`](../adr/README.md) is the whole convention, with no
exception attached: an ADR is an immutable record, and a decision that changes is replaced by a new
`accepted` ADR while the old one is marked `superseded`. That is the ADR form as
[MADR](https://adr.github.io/madr/) defines it, and it is what
[ADR-0000](../adr/0000-record-architecture-decisions.md) decided.

If your project wants the in-place regime instead — a legitimate choice for a design document that
is shipped rather than lived — record that as your own decision, in your own ADR. Do not infer it
from the fact that the upstream did it.

## Inbound pointers

Every reference into this file, for the rename sweep and as the removal checklist. Each is one line
carrying `boilerplate-only:line`.

| File | Location |
| --- | --- |
| [`docs/adr/README.md`](../adr/README.md) | last bullet of *Conventions* |
| [`docs/ja/adr/README.ja.md`](../ja/adr/README.ja.md) | 「規約」の最終項目 |

This file itself is removed by path, not by marker.
