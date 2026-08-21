# QueryService DML

English | [日本語](README.ja.md)

Read-only SQL for search and list optimization, bypassing the domain layer.

## Purpose

- Provide optimized read-only queries with JOINs, aggregation, and filtering at the SQL level.
- Separate read concerns from write operations and transaction management.
- Generate type-safe Go code via sqlc for compile-time parameter and scan validation.

## Predicates That Mirror a Domain Invariant

Some predicates restate, in SQL, a condition an aggregate already guarantees. `canceled_at IS NULL`
is the negation of `Purchase.IsCanceled()` (`status == StatusCanceled`), and `published_at IS NOT NULL`
is `product.IsPublished()`. The two forms stay equivalent because the aggregate validates that
correspondence when it is reconstructed — `(status == StatusCanceled) != (canceledAt != nil)` in
`internal/domain/purchase`.

The definition lives on the domain method, not in the query. A query's comment therefore names which
method its predicate mirrors and stops there; when the domain rule changes, this section and the
method move together and the queries need no edit.

## Infrastructure Mapping

Implementation: `internal/infrastructure/rdb/query_service/`

## Directory Structure

One directory per read model, named after the aggregate the projection is read from.

## Naming Convention

- Files: verb + target (e.g., `list_published_products.sql`)
- `-- name:` annotation required on all queries

## Code Generation

```sh
make gen-query
```
