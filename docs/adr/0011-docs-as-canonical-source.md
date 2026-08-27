---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, docs]
---

# ADR-0011: Docs-as-canonical-source strategy

## Status

accepted

## Context

The project documentation serves two distinct audiences with different needs:

- **AI agents** — need precise, unambiguous English technical prose to generate correct code
  and follow correct workflows. They must read a single authoritative source; reading
  translations introduces translation-lag risk. Additionally, AI tools typically consume
  approximately 1.5–2.0× more tokens when processing Japanese text and carry a higher risk of
  misreading and hallucination, making English the more reliable input language for agents.
- **Human developers** — benefit from a navigable portal rather than raw markdown files.

Structural concerns arise:

1. If the portal is the primary documentation surface rather than the source files, portal
   generation failures block documentation access and the portal format becomes a constraint
   on content.
2. If documentation is scattered without a stable directory convention, generators
   (`scripts/portal/gen-docs-json.ts`) and agent harnesses (`AGENTS.md`) cannot reliably locate
   canonical content.

<!-- boilerplate-only:begin -->
A further concern is about who the documentation is for. Every language kept here is inherited
wholesale by the projects built from this scaffold, along with the cost of maintaining it. A team
that reads only one of the two would carry the other for nobody. So "which languages does this
documentation have" has two answers, and only the first of them is this repository's to give.
<!-- boilerplate-only:end -->

## Decision

Adopt a two-layer documentation strategy:

1. **Canonical markdown** — `docs/**/*.md` (excluding `docs/portal/**`) is the authoritative
   source. These are what agents read, what rules reference, and what the portal renders.
   The documentation is maintained in a single language.
2. **Generated portal** — `docs/portal/docs.json` and `docs/portal/guides/**` are generated
   by `scripts/portal/gen-docs-json.ts` from the canonical sources, driven by
   `docs/portal/manifest.yaml`. Portal content must not be edited manually.

<!-- boilerplate-only:begin -->
**The pair belongs to this repository, not to what is built from it.** Which languages a project
keeps is its own decision, and `make setup-remove-doc-language` folds the documentation and the
skills down to one of them — deleting the translations, or renaming them onto the canonical names
so the filename contract (`SKILL.md`, `README.md`) still holds. The checks that require a pair
(`doc-ref-lint`, `skill-lint`) come out with it: a rule with nothing left to check is a rule that
only fails.
<!-- boilerplate-only:end -->

`docs/portal/manifest.yaml` is the structural control file for the portal: it maps source
files to portal destinations, defines the navigation hierarchy (groups, sections,
subgroups), and controls section display titles. It is human-maintained.

The following directories are reserved by the generator and cannot be used as normal
documentation sections: `docs/portal`, `docs/openapi`, `docs/coverage`, `docs/er-diagram`.

## Consequences

### Positive Consequences

- AI agents have a single unambiguous reading target (English canonical); the rule is
  enforceable in agent harnesses.
- Portal content is always reproducible from source; `docs/portal/docs.json` is a generated
  artifact and can be regenerated on any machine or CI runner.
- Adding a new documentation section follows a clear convention: a `docs/<section>/` directory,
  and an entry in `docs/portal/manifest.yaml`.

### Negative Consequences

- `docs/portal/manifest.yaml` must be updated when new sections are added or renamed; an
  out-of-date manifest causes new docs to not appear in the portal.
- The reserved directory list limits naming choices for new documentation sections.

## Alternatives Considered

<!-- boilerplate-only:begin -->
### Fixing the language set here

Maintain the pair and hand it on as a settled property. Rejected: it makes every project built
from this one pay for a language it may never read, and the payment recurs — each document edited
twice, indefinitely. It is also the more expensive of the two possible errors: adding a translation
later is ordinary work, while removing 400 of them together with the checks and the portal wiring
that assume they exist is not.

<!-- boilerplate-only:end -->
### Portal as primary source

Maintain documentation in the portal's native format and derive markdown from it. Rejected:
the portal is a rendering concern; making it the canonical source ties content to a specific
rendering pipeline and complicates direct agent consumption.

## Notes

- Source: [`docs/index.md`](../index.md) — documentation overview and recommended reading
  order for humans and agents.
- Source: [`docs/maintenance/docs-structure.md`](../maintenance/docs-structure.md) — full
  structure rules for the documentation portal generator.
- Source: [`docs/portal/manifest.yaml`](../portal/manifest.yaml) — portal navigation and
  source-to-destination mapping.
- Agent reading rule: `AGENTS.md` §§ "Canonical Documentation" and "Documentation scope for
  agents".
