---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, search]
---

# ADR-0028: Run full-text search in the database using a GENERATED STORED column with a GIN trgm index

## Status

accepted

## Context

The user search API requires keyword matching across multiple user fields: first name, last
name, email, phone, city, street, building, and postal code. Several implementation
strategies exist:

- **Application-layer filtering**: fetch all rows and filter in Go. Unacceptably slow at
  scale; no index benefit.
- **Multiple `ILIKE` conditions on individual columns**: produces complex SQL with a
  disjunctive WHERE clause across eight columns; each column requires a separate index
  pass; no single index can cover all columns efficiently.
- **External search engine** (e.g., Elasticsearch, OpenSearch): best-in-class relevance and
  NLP capabilities but introduces an additional infrastructure dependency, an
  eventually-consistent sync mechanism, and significant operational overhead.
- **PostgreSQL `tsvector` / `tsquery`**: language-aware full-text search with stemming and
  ranking. Powerful but requires language dictionary configuration and does not support
  arbitrary substring matching well.
- **PostgreSQL pg_trgm with a computed column**: trigram-based substring search indexed via
  GIN. Supports `ILIKE` efficiently, requires no language configuration, and runs entirely
  within the existing PostgreSQL instance.

The dataset size and current search requirements do not warrant an external search engine
or NLP-grade language processing. Efficient substring matching across all relevant fields
is sufficient.

## Decision

A `GENERATED ALWAYS AS (...) STORED` column named `search_text` is added to the `users`
table (migration `000011_users_table_search_text_column.up.sql`). It concatenates the
relevant fields with spaces at write time:

```sql
search_text TEXT GENERATED ALWAYS AS (
    COALESCE(first_name, '') || ' '
    || COALESCE(last_name, '') || ' '
    || COALESCE(email, '') || ' '
    || COALESCE(phone, '') || ' '
    || COALESCE(city, '') || ' '
    || COALESCE(street, '') || ' '
    || COALESCE(building, '') || ' '
    || COALESCE(postal_code, '')
) STORED
```

A GIN index using `gin_trgm_ops` is built on `search_text`:

```sql
CREATE INDEX users_search_text_trgm_idx ON users
USING gin (search_text gin_trgm_ops);
```

Keyword search queries match against `search_text` using `ILIKE ANY(patterns::TEXT[])` and
are executed via QueryService (see [ADR-0025](0025-lightweight-cqrs.md)), not Repository,
because the user search crosses the `users`+`prefectures` aggregate boundary and returns a
view-specific DTO.

## Consequences

### Positive Consequences

- The generated column is maintained automatically by PostgreSQL; application writes do not
  need to update the search field separately.
- `ILIKE ANY` with a GIN trgm index provides efficient substring matching for typical
  dataset sizes without language configuration.
- No additional infrastructure: the feature is entirely within PostgreSQL.
- The single `search_text` column simplifies the query: one `ILIKE ANY` condition replaces
  eight disjunctive conditions.

### Negative Consequences

- `search_text` is a denormalized concatenation. Adding a new searchable field requires a
  migration to alter the generated column expression and rebuild the GIN index.
- pg_trgm `ILIKE` is not relevance-ranked; all matches are treated equally.
- pg_trgm does not support stemming, synonym expansion, or language-aware tokenization.
- The GIN index increases write overhead and storage proportional to the size of
  `search_text`.

## Alternatives Considered

### Multiple ILIKE conditions on individual columns

No schema change needed. However, a disjunctive WHERE across eight columns cannot be
covered by a single index, forcing index-scan-and-filter for each column. Rejected for
performance reasons at non-trivial row counts.

### External search engine (Elasticsearch, etc.)

Best relevance and NLP capabilities. Adds an infrastructure dependency, an eventual
consistency sync layer, and operational burden (index management, cluster health). Rejected
as premature — the current requirements do not justify the complexity.

### PostgreSQL tsvector / tsquery

Language-aware full-text search with GIN indexing. Supports stemming and `ts_rank`
relevance scoring. Requires dictionary configuration for Japanese content and does not
support arbitrary substring matching as naturally as trgm `ILIKE`. Rejected because
substring matching is the dominant use case and trgm requires no language setup.

## Notes

- Source: [`database/migrations/000011_users_table_search_text_column.up.sql`](../../../database/migrations/000011_users_table_search_text_column.up.sql).
- Source: [`database/dml/query_service/user/select_users_by_keyword.sql`](../../../database/dml/query_service/user/select_users_by_keyword.sql).
- Related: [ADR-0025](0025-lightweight-cqrs.md) (QueryService is used for this search
  because it crosses aggregate boundaries).
