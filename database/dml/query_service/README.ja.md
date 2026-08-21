# QueryService DML

[English](README.md) | 日本語

検索・一覧取得の最適化を目的とした読み取り専用SQLを管理します。

## 目的

- JOIN・集計・フィルタリングをSQLレベルで最適化し、高速な読み取り専用クエリを提供します。
- 書き込み処理やトランザクション管理から参照処理を分離します。
- sqlc によるコード生成で、パラメータやスキャンの型をコンパイル時に検証します。

## ドメインの不変条件を写した述語

一部の述語は、集約が既に保証している条件を SQL で言い直したものである。`canceled_at IS NULL` は
`Purchase.IsCanceled()`（`status == StatusCanceled`）の否定であり、`published_at IS NOT NULL` は
`product.IsPublished()` に当たる。両者が等価であり続けるのは、集約が再構築時にその対応を検証するため
である（`internal/domain/purchase` の `(status == StatusCanceled) != (canceledAt != nil)`）。

定義を持つのはドメインのメソッドであってクエリではない。したがってクエリのコメントは、その述語が
どのメソッドを写したものかを名指すに留める。ドメイン規則が変われば、この節とメソッドが一緒に動き、
クエリ側は編集を要しない。

## インフラストラクチャマッピング

実装: `internal/infrastructure/rdb/query_service/`

## ディレクトリ構成

読み取りモデルごとに 1 つのディレクトリを置き、投影の起点となる集約の名前を付ける。

## 命名規則

- ファイル名: 動詞 + 対象名（例: `list_published_products.sql`）
- 全てのクエリに `-- name:` アノテーションが必須

## コード生成

```sh
make gen-query
```
