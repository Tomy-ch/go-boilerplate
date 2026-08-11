---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, codegen]
---

# ADR-0024: sqlcによる型安全なSQLアクセスの生成

English canonical: [0024-sqlc-type-safe-sql.md](../../adr/0024-sqlc-type-safe-sql.md)

## ステータス

accepted

## 背景

SQLファーストの方針（[ADR-0023](0023-sql-first-data-access.ja.md)）を前提とすると、手書きのSQLをGoに**コンパイル時の型安全な**コードとして届ける必要があるが、ORMのランタイム抽象や暗黙的なクエリ生成を再導入したくない。

## 決定

**sqlc**を使用してSQLクエリからGoコードを生成する。開発者がSQLを記述し、sqlcが型付きGo関数と行構造体を生成し、インフラレイヤーがそれをラップする。

## 影響

### ポジティブな影響

- SQLとGoの間のコンパイル時型安全性。
- SQLがソース成果物として明示的かつ読みやすいまま保持される。
- ランタイム抽象が最小限 — 生成コードは薄い。

### ネガティブな影響

- ワークフローにコード生成ステップ（`make gen-query`）が必要であり、生成ファイルは手動で編集しない。
- クエリ機能はsqlcが解析・マッピングできる範囲に限定される。

## 検討した代替案

### GORM

却下：便利なORMだが、SQLファーストの決定が避けようとしているORM抽象と暗黙的なクエリ生成を再導入する。

### Ent

却下：SQLを直接記述するのとは異なる開発ワークフローを強制するスキーマファーストのアプローチ。

## 補足

- [ADR-0023](0023-sql-first-data-access.ja.md)（SQLファースト）を前提とする。
- 生成コードは手動編集不可 — [`docs/rules.md`](../rules.ja.md#生成コードルール)の生成コードルール参照。
- かつての `docs/decisions.md` から移行。
