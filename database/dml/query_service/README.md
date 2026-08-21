# QueryService DML

English | [日本語](README.ja.md)

Read-only SQL for search and list optimization, bypassing the domain layer.

## Purpose

- Provide optimized read-only queries with JOINs, aggregation, and filtering at the SQL level.
- Separate read concerns from write operations and transaction management.
- Generate type-safe Go code via sqlc for compile-time parameter and scan validation.

## Predicates That Mirror a Domain Invariant

<!-- sample-api:replace-begin -->
Some predicates restate, in SQL, a condition an aggregate already guarantees. `canceled_at IS NULL`
is the negation of `Purchase.IsCanceled()` (`status == StatusCanceled`), and `published_at IS NOT NULL`
is `product.IsPublished()`. The two forms stay equivalent because the aggregate validates that
correspondence when it is reconstructed — `(status == StatusCanceled) != (canceledAt != nil)` in
`internal/domain/purchase`.
<!-- sample-api:replace-with -->
<!-- = Some predicates restate, in SQL, a condition an aggregate already guarantees. The two forms stay -->
<!-- = equivalent because the aggregate validates that correspondence when it is reconstructed. -->
<!-- sample-api:replace-end -->

<!-- 撤去後にこの箇所へ自分の例を置くための指針。
     目的: 具体の対応が 1 組も無いと、どの述語がどのメソッドを写したものかを読み手が判別できない。
     意義: 効くのは「SQL の述語と、それを保証するドメインメソッドが 1 対 1 で名指せること」。
     書き方: 自分の集約で、列に対する条件とそれを保証するメソッド名を 1 組挙げ、
             再構築時にその対応を検証している箇所を指す。 -->

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
