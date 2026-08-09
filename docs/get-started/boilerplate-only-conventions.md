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

## Arguments that rest on the sample feature set

This repository ships a sample feature set that is developed, harvested, and then removed. Several
statements elsewhere borrow their force from it. A fork has no sample set, so what it inherits is the
rule, never the illustration.

### Sample occupants kept to make a rule legible

The only provider in `internal/infrastructure/rdb/command_service/` belongs to the sample purchase
feature, and it is the sole registration in the `command_service` sub-module of `persistenceModule`
(`internal/di/module/persistence.go`). Removing the samples empties the sub-module and leaves
[ADR-0028](../adr/0028-lightweight-cqrs.md)'s CommandService section describing an intended design
with no occupant — which is the state a fork starts from.

The upstream keeps that occupant deliberately: the eligibility bar ADR-0028 states (which writes
deserve a CommandService) is only legible against a concrete case that meets it, so the sample
carries the bar's reading.

**This deviation does not transfer.** In a project built from the template, what goes into
`command_service` is decided by that project's requirements. Zero occupants is not a defect, and
"keep an implementation so the bar can be read" is not a reason that project holds.

### The sample API moves the authorization gate's boundary

Which environments the environment gate in `internal/di` (`provideAuthorizer`) names is moved by the
presence of the sample API. With the sample present it wires the allow-all authorizer for CI / test
and the `user_roles` authorizer for local through production; after `make setup-remove-sample-api`
the `user_roles` case is removed, leaving local / CI / test on allow-all and every production-like
environment fail-closed until a real RBAC / policy adapter is wired.

[`internal/di/README.md`](../../internal/di/README.md) keeps only the general form — adding or
removing an authorization implementation moves the boundary, so read the `switch` — because that
prose survives the sample removal. The concrete trigger lives here instead.

**This deviation does not transfer.** A project built from this repository has no
`make setup-remove-sample-api`, and the "with / without the sample" distinction disappears with it.
What moves the boundary there is that project adding or dropping an authorization implementation,
not a setup step.

### Broker-SDK isolation is checked by running the sample removal

[`internal/infrastructure/queue/sqs/README.md`](../../internal/infrastructure/queue/sqs/README.md)
states the condition a fork inherits: `github.com/aws/aws-sdk-go-v2/service/sqs` enters
`go list -deps ./cmd/` only when wiring that selects the adapter is present. That is a claim about
the link graph, and it holds however the repository is distributed.

How the upstream *observes* it does not. Here the wiring that selects the adapter belongs to the
removable sample set, so the condition is checked by running the sample removal and comparing the
dependency graph on either side — a procedure `.github/workflows/sample-removal-check.yaml` repeats
on every pull request.

**This does not transfer.** A project built from the template has no sample set to remove and no
pre-sample state to compare against. The condition survives; the removal was only how the upstream
happened to watch it. Do not read the check as evidence that the invariant is about removal — it is
about which wiring pulls the SDK in.

## Arguments that rest on there being no adopter yet

The remaining deviations share one shape: the upstream stops somewhere, and the reason it stops
there is that the party who would settle the question does not exist yet. In a fork that party
exists, so the reason is gone and the stopping point is no longer justified by it.

### Setup scripts: why the pure-module split is a rule, not a convention

[ADR-0078](../adr/0078-scripts-in-node-go.md) states the general form: every script keeps its
decision logic in a pure module with a test suite next to it, because a gate whose failure mode is
to inspect nothing and still exit `0` can be pinned by a type checker and a test and by nothing
else. The one-time setup scripts under `scripts/setup/` are what turned that from a convention into
a rule, and the argument is upstream-only.

When a replacement rule over-matches or misses a file type, the person who finds out is someone who
cloned the repository minutes ago, holds no context to debug with, and is looking at their own
project for the first time. Several of those scripts are never executed by CI at all, and every one
of them rewrites that person's repository — a Go module path across every file in the tree, the
LICENSE holder, the owner field of every CODEOWNERS rule.

The same relation bounds what the tests are worth. The self-deleting setup tools take their tests
with them, so those tests protect this repository's own CI rather than the one created from it.

**This premise does not transfer.** In a project built from the template the setup scripts have
already run — there is no stranger holding no context, and no CI whose coverage stops at the moment
of use. What survives is the general rule in ADR-0078, which stands on the gate failure mode alone.

### Why the domain Module rules stop at the mechanical floor

[`internal/domain/README.md`](../../internal/domain/README.md) states the general form: the naming
and structure rules for a domain package are mechanical — they say what to call things, not what a
division should reveal — and where a real domain is present those rules are the floor, with the
model-revealing divisions added on top. That is what a fork inherits.

What the general form no longer says is *why the upstream stops at the floor*. A template has no
real domain to have an insight about, so the only lines it can honestly draw are the ones the
architecture implies. Evans expects a Module's boundaries and names to carry an insight about the
domain and to evolve as the model does; this repository can supply neither, because the model it
would evolve against does not exist here. The sample aggregates illustrate the mechanics; they are
not a domain anyone reasoned about.

**This premise does not transfer.** A project built from the template has a real domain, and the
reason for stopping at the mechanical floor is gone with it. Do not read the upstream's silence
about model-revealing divisions as a position that they are unnecessary — read it as the upstream
having nothing to say them about.

### `未確定` as a terminal state, not an open task

[`docs/design/context-map.md`](../design/context-map.md) records an edge whose settling fact is not
available as `未確定`, carrying its evidence and the question that would settle it. That is the
general form, and it is what a fork inherits.

While this repository is the upstream boilerplate, **most of those edges stay `未確定` permanently,
and that is the correct state rather than an omission.** What would settle a downstream edge —
whether the upstream will take our requirements — is a fact about the *adopting* organisation, and
no such organisation exists yet. There is nobody to ask, so the map records the question instead of
guessing at an answer.

**This does not transfer.** A project built from the template has real counterparties on the other
side of each edge, and the organisational fact is one somebody there can go and establish. An edge
left `未確定` in that repository is not a correct terminal state; it is a question nobody has asked
yet.

## What applies after setup

Once this file is gone, [`docs/adr/README.md`](../adr/README.md) is the whole convention, with no
exception attached: an ADR is an immutable record, and a decision that changes is replaced by a new
`accepted` ADR while the old one is marked `superseded`. That is the ADR form as
[MADR](https://adr.github.io/madr/) defines it, and it is what
[ADR-0000](../adr/0000-record-architecture-decisions.md) decided.

If your project wants the in-place regime instead — a legitimate choice for a design document that
is shipped rather than lived — record that as your own decision, in your own ADR. Do not infer it
from the fact that the upstream did it.

## Finding the pointers

There is no list of them here, and there was one until the removal learned to scan. A list is a
second place to be right, and the way it goes wrong is silent: a pointer written outside it is
simply never removed, while the pass still reports success. The pass reads the repository instead,
so what is authoritative is the markers themselves:

```sh
grep -rn "boilerplate-only:" --exclude-dir={.git,node_modules,vendor} .
```

This file itself is removed by path rather than by marker, which is why it can hold the premise
whole instead of in fragments.
