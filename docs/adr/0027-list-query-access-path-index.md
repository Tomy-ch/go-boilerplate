---
status: accepted
date: 2026-08-06
deciders: [maintainers]
tags: [persistence, performance, migration]
---

# ADR-0027: Index a filtered, ordered list query along the access path the planner actually takes

## Status

accepted

## Context

A list endpoint that filters rows, orders them, and returns only the first `limit` of them is a
recurring shape in this project. Without an index whose key order matches the requested order, the
database reads every matching row, sorts the whole set, and discards all but the first page — the
`LIMIT` cannot short-circuit anything, so response time grows with the table rather than with the
page.

Adding an index is not automatically the fix. An index only removes the sort if the planner can walk
it *in the requested order while applying the filter*, and whether it can depends on how the filter is
expressed, not only on which columns are indexed. Two cases in this repository make the difference
concrete:

- A filter comparing two columns of the same row (`quantity <= stock_warning_threshold`) can never be
  an index key, because an index stores column values, not relationships between them. It can still
  be evaluated as a residual condition while walking an index ordered by the requested sort columns.
- A filter expressed against a joined table (`purchase_statuses.code = ?`) resolves to a single value
  of the local foreign-key column, but the planner is not obliged to use that value as an ordered
  index condition. It may instead choose a bitmap scan, which loses the index's ordering and
  therefore reintroduces the sort over every matching row.

The second case is the one that makes the naive rule dangerous: the index is present, the sequential
scan disappears from the plan, and the query is no faster, because the bitmap scan still materializes
every matching row before the top-N sort. The table then pays the index's write cost for nothing.

## Decision

An index is added for a filtered, ordered list query **only when an `EXPLAIN` at representative scale
shows that it changes the plan** — specifically, that the sort node is gone and the scan stops early
at the `LIMIT`.

The index's key order must equal the query's `ORDER BY`. A predicate that cannot be an index key
(a column-to-column comparison) is left as a residual condition; a predicate that merely narrows which
rows are worth indexing at all (a nullable configuration column) becomes the index's `WHERE` clause,
so the index holds only the operationally relevant subset.

**An index that leaves the plan unchanged is rejected rather than kept "in case it helps later".** Its
cost is real and continuous — every write to an indexed column maintains it — while its benefit is
zero until the query is rewritten. Where the measurement shows that the query's *form* is what
prevents the index from working, changing that form is a separate decision about the query, not a
reason to keep an inert index alongside it.

This applies to the physical access path only. It does not license changing what a query returns or
how a domain rule is expressed in SQL.

## Consequences

### Positive Consequences

- A qualifying list query costs the page it returns rather than the table it scans: the index is
  walked in the requested order and abandoned once `LIMIT` rows are produced.
- A partial index is sized by the operationally relevant subset rather than by the table, so a column
  that is set on a minority of rows does not cost a full-table index.
- Rejecting an inert index keeps the write path free of maintenance that buys nothing, and leaves the
  real obstacle visible instead of appearing to have addressed it.

### Negative Consequences

- Indexing a column that the application updates makes that column's writes more expensive, and
  disqualifies the row from a heap-only update. Where the indexed column is the one a write endpoint
  exists to change, this trades a cost on the write path — often the busier one — for a read that may
  be occasional.
- A residual predicate is evaluated per row after the index lookup, so an index ordered by the sort
  columns still reads and discards non-matching rows. When matches are sparse, filling one page can
  require walking a large part of the index.
- The correspondence between a query's `ORDER BY` and its index's key order is not machine-checked.
  Reordering the query silently restores the sort: the tests stay green and only the performance
  regresses.
- Each index adds storage and vacuum/analyze work, which a fork that never calls the endpoint pays for
  without benefit. Such a fork can drop it in a migration of its own.

## Alternatives Considered

### Index every filtered list query on principle

Uniform and requires no measurement. Rejected: it produces indexes that do not change the plan, which
are pure cost. The measurement is what separates the two cases, so it cannot be skipped.

### Index nothing, and leave physical design to the consuming project

Defensible for a boilerplate, since the right index depends on data volume and write mix that only the
consuming project knows. Rejected: a list query whose filter axis has no index is a misleading example
regardless of scale, and the reader cannot tell an unmade decision from a deliberate one.

### Add covering columns (`INCLUDE`) so the residual predicate is evaluated in the index

Removes heap fetches for the discarded rows. Rejected: it widens every index entry on a column the
write path already updates, deepening the write-side cost this ADR treats as the primary trade-off,
and the narrowing already achieved by the partial predicate makes the remaining gain small.

### Rewrite the predicate as an expression index

A column-to-column comparison becomes indexable when rewritten as an expression over the difference.
Rejected: it changes the query's order to the expression's order, which is a different result from the
one the endpoint specifies, and it moves a domain rule's SQL form away from the domain method that
defines it.

### Materialize the predicate as a generated column and index that

Would let the index satisfy both the filter and the order. Rejected: it is a schema change that
propagates into generated models, the domain entity, and the API contract — a much larger change than
the access path requires.

## Notes

- Applied in [`database/migrations/000010_create_products.up.sql`](../../database/migrations/000010_create_products.up.sql)
  for [`database/dml/repository/product/select_low_stock_products.sql`](../../database/dml/repository/product/select_low_stock_products.sql).
- Related: [ADR-0021](0021-sql-first-data-access.md) (SQL-first access);
  [ADR-0024](0024-append-only-immutable-migrations.md) (migrations are append-only);
  [ADR-0033](0033-uuidv7-identifiers.md) (identifier ordering and B-tree behavior).
