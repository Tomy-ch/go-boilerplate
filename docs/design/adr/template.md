---
status: proposed        # proposed | accepted | superseded | deprecated
date: YYYY-MM-DD
deciders: []            # who made the call, e.g. [maintainers]
supersedes:             # ADR number this replaces, if any (e.g. 0003)
superseded-by:          # ADR number that replaces this one, if any
tags: []                # e.g. [architecture, http, persistence]
---

# ADR-NNNN: imperative decision title

## Status

proposed | accepted | superseded by [ADR-XXXX](XXXX-....md)

## Context

What forces are at play — the problem, constraints, and goals that make a decision
necessary. State the *why*, not the solution. (For an **exclusion** ADR, state what
capability is being deliberately left out and the pressure to include it.)

## Decision

The choice, stated in one or two sentences. For an exclusion: "We deliberately do NOT
provide X." Be specific about scope and boundary.

## Consequences

### Positive Consequences

- ...

### Negative Consequences

- ...

<!-- Optional: ### Neutral Consequences -->

## Alternatives Considered

### Alternative A

Why it was weighed and why it was rejected.

### Alternative B

...

## Notes

Links to design docs, rules that enforce this decision (`docs/rules.md#...`), related
ADRs. Optional.
