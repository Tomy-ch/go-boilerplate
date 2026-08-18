---
status: accepted
date: 2026-08-17
deciders: [maintainers]
tags: [contract, openapi]
---

# ADR-0018: Repeat the parameter name for multi-value filters, and keep free-text input scalar

## Status

accepted

## Context

A search endpoint carries two kinds of query parameter that look alike and behave nothing
alike: a **filter that accepts several values for one condition** (categories, statuses,
grouping units) and a **free-text field the user typed** (a keyword). Both end up in the
query string, so a single convention has to cover both or endpoints drift apart.

OpenAPI offers more than one serialization for an array in a query, selected by `style` and
`explode`. The two candidates for `style: form` are the default `explode: true`
(`categoryCodes=1&categoryCodes=2`) and `explode: false` (`categoryCodes=1,2`). They are not
interchangeable, and a parameter cannot accept both: `explode` is a single boolean, and both
layers of this repository's request path reject the form they were not declared for.

Three forces decide between them.

**The URL has a practical ceiling.** Front proxies and CDNs commonly cap the request line
around 8 KB. Repeating the name costs `len(name) + 1` per value, while a delimiter costs it
once per condition. For a 32-value `int` code filter these compute to 639 and 205 characters
respectively; against an 8 KB budget that is roughly 12 versus 40 multi-value conditions on
one endpoint.

**A delimiter cannot be escaped.** Both `kin-openapi` (request validation) and the
`oapi-codegen` runtime split the value *after* percent-decoding, so a `%2C` a client sends
inside a value is indistinguishable from a separator by the time either sees it. A
delimiter-joined array therefore cannot carry a value containing the delimiter at all —
switching to `spaceDelimited` or `pipeDelimited` only moves the collision to another
character.

**Free text dominates the budget regardless.** Percent-encoded UTF-8 costs up to 9
characters per character, so one `maxLength: 255` text parameter can reach 2,295 characters
— an order of magnitude more than the 434-character gap between the two array forms.

(The lengths above are computed from the declared `maxItems` / `maxLength` bounds, not
measured against a running proxy.)

## Decision

Search query parameters take one of two shapes, and nothing in between.

| Shape | Declared as | Wire form | Used for |
| --- | --- | --- | --- |
| Multi-value filter | `type: array` with OpenAPI's default serialization (`style: form`, `explode: true`) | `categoryCodes=1&categoryCodes=2` | a reference to a master row (`code`) or a fixed `enum` |
| Free-text input | single `string` with `maxLength` | `keyword=...` | text the user typed |

Three rules follow from the table:

1. **A condition that accepts several values repeats its name.** The array is declared with
   `uniqueItems` and `maxItems` as wire limits.
2. **Only a row reference or a fixed `enum` may be multi-valued.** Free-text input is never
   an array.
3. **A search that outgrows this moves to a `POST` body** — many facets, or values too large
   for a URL. The answer is not a longer query string.

Rule 2 is what makes rule 1 affordable: because every array element is an ASCII code or an
enum member, neither the delimiter collision nor the UTF-8 blow-up can reach an array, and
the cost of repeating a name stays bounded by `maxItems`.

Which identifier a filter accepts is settled separately by
[ADR-0030 (master-data-via-migration)](0030-master-data-via-migration.md): a client sends the
master row's `code`, never its UUID and never its display name. That decision is what keeps
Japanese text out of a filter parameter in the first place.

## Consequences

### Positive Consequences

- The wire form is OpenAPI's default, so no client generator has to implement a
  non-default path. `explode: false` is a documented source of gaps in several
  `openapi-generator` languages; this repository stops depending on that path.
- The request-validation layer and the binding layer agree. `kin-openapi` reads only the
  first occurrence of a non-exploded parameter while the `oapi-codegen` runtime rejects
  extra occurrences outright — a divergence that exists **only** on the non-exploded path
  and disappears here.
- A value containing a comma is representable, so a future filter is not silently
  constrained by a serialization choice made for today's integer codes.
- The rules are checkable while reading a spec file: an array of `string` with no `enum` is
  a violation on sight.

### Negative Consequences

- The same filter costs a longer URL. On one endpoint the ceiling drops from roughly 40
  multi-value conditions to roughly 12; past that the endpoint must become a `POST` search
  under rule 3.
- Rules 2 and 3 have no mechanical gate. They are enforced by spec review, so a
  free-text array can be introduced by an author who has not read this record.

## Alternatives Considered

### A delimiter-separated array (`style: form`, `explode: false`)

Rejected. It buys URL headroom that rule 3 makes unnecessary, and it pays for that headroom
with an unescapable delimiter, a non-default path through client generators, and a
disagreement between this repository's two request layers. Its headroom advantage is also
second-order: free-text parameters dominate the budget either way.

### Bracket notation (`categoryCodes[]=1&categoryCodes[]=2`)

Rejected. This is the PHP / Rails convention and jQuery's `$.param()` default, but OpenAPI
has no `style` for it — expressing it means writing the brackets into the parameter *name*
and letting tooling treat them as opaque. It is also the longest of the three forms, since
`[]` percent-encodes to six characters on every occurrence.

### Matrix parameters or path segments (`/products;categoryCodes=1,2`)

Rejected. OpenAPI's `matrix` and `label` styles apply to **path** parameters, and a path
parameter must be `required: true`. Optional filters cannot be expressed there at all.

### Abbreviating parameter names (`cc=1&cc=2`)

Rejected. Shortening the name is the only way repetition becomes materially cheaper, and it
is not worth it: `cc` reads as "carbon copy" or "credit card" to anyone outside this
repository, it contradicts the ubiquitous language the master-data vocabulary establishes,
and it would set an abbreviated public-API name as the convention every later endpoint
follows. The saving is second-order against rule 3's ceiling.

## Notes

- Convention and examples: [`openapi/components/parameters/README.md`](../../openapi/components/parameters/README.md).
- Identifier choice for master-data filters: [ADR-0030 (master-data-via-migration)](0030-master-data-via-migration.md).
- The runtime enforcement referred to above is
  [ADR-0015 (spec-driven-request-validation)](0015-spec-driven-request-validation.md); which
  layer owns a boundary value is [ADR-0017 (boundary-value-ownership)](0017-boundary-value-ownership.md).
