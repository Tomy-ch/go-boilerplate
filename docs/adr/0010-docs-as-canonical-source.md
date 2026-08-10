---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [foundational, docs]
---

# ADR-0010: Docs-as-canonical-source strategy (English canonical + ja mirror + portal)

## Status

accepted

## Context

The project documentation serves two distinct audiences with different needs:

- **AI agents** — need precise, unambiguous English technical prose to generate correct code
  and follow correct workflows. They must read a single authoritative source; reading
  translations introduces translation-lag risk. Additionally, AI tools typically consume
  approximately 1.5–2.0× more tokens when processing Japanese text and carry a higher risk of
  misreading and hallucination, making English the more reliable input language for agents.
- **Human developers** — may prefer Japanese; benefit from a navigable portal rather than
  raw markdown files.

Three structural concerns arise:

1. If English and Japanese documents are maintained in parallel without a clear primacy rule,
   they diverge. A diverged translation misleads whichever audience reads it.
2. If the portal is the primary documentation surface rather than the source files, portal
   generation failures block documentation access and the portal format becomes a constraint
   on content.
3. If documentation is scattered without a stable directory convention, generators
   (`scripts/portal/gen-docs-json.ts`) and agent harnesses (`AGENTS.md`) cannot reliably locate
   canonical content.

## Decision

Adopt a three-layer documentation strategy:

1. **English canonical** — `docs/**/*.md` (excluding `docs/ja/**` and `docs/portal/**`) is
   the authoritative source. These are what agents read, what rules reference, and what the
   portal renders.
2. **Japanese mirror** — `docs/ja/**/*.ja.md` files are human-maintained translations of the
   English canonical docs. They are never read by agents (per `AGENTS.md`: agents must not
   read `*.ja.md` files). Naming convention: `<name>.ja.md` in a parallel directory structure
   under `docs/ja/`.
3. **Generated portal** — `docs/portal/docs.json` and `docs/portal/guides/**` are generated
   by `scripts/portal/gen-docs-json.ts` from the canonical sources, driven by
   `docs/portal/manifest.yaml`. Portal content must not be edited manually.

`docs/portal/manifest.yaml` is the structural control file for the portal: it maps source
files to portal destinations, defines the navigation hierarchy (groups, sections,
subgroups), and controls section display titles. It is human-maintained.

The following directories are reserved by the generator and cannot be used as normal
documentation sections: `docs/portal`, `docs/openapi`, `docs/coverage`, `docs/er-diagram`,
`docs/ja`.

## Consequences

### Positive Consequences

- AI agents have a single unambiguous reading target (English canonical); the rule is
  enforceable in agent harnesses.
- Translation lag is bounded: the Japanese mirror may trail behind English updates, but the
  English canonical is always authoritative.
- Portal content is always reproducible from source; `docs/portal/docs.json` is a generated
  artifact and can be regenerated on any machine or CI runner.
- Adding a new documentation section follows a clear convention: an English `docs/<section>/`
  directory, a parallel `docs/ja/<section>/` directory, and an entry in
  `docs/portal/manifest.yaml`.

### Negative Consequences

- Maintaining two language versions of every document is labor-intensive; Japanese mirrors
  can lag behind English canonical updates.
- `docs/portal/manifest.yaml` must be updated when new sections are added or renamed; an
  out-of-date manifest causes new docs to not appear in the portal.
- The reserved directory list limits naming choices for new documentation sections.

## Alternatives Considered

### Single-language (English only)

Drop Japanese translations entirely. Simpler maintenance, but reduces accessibility for the
Japanese-speaking development teams who are this documentation's primary human readers.

### Auto-translation

Generate Japanese translations automatically from English. Rejected: machine-translation
quality is insufficient for precise technical content; a mistranslated architectural rule
can mislead developers.

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
