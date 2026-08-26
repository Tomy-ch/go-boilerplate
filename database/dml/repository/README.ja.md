# Repository DML

[English](README.md) | 日本語

単一 Aggregate の永続化と単純な読み取り（CRUD、および Aggregate 自身の属性による単純なフィルタ・一覧・件数）に使用するSQLを管理します。

## 目的

- ドメインモデルの永続化・取得に必要な INSERT / UPDATE / DELETE / SELECT クエリ（単一 Aggregate 自身の属性による単純なフィルタ・一覧・件数を含む）を集約します。
- ビジネスロジックを含まないシンプルなSQLを維持します。Aggregate を横断する読み取り・集計・複雑な結合は QueryService で行います。
- sqlc によるコード生成で、型安全なリポジトリ実装を提供します。

## インフラストラクチャマッピング

実装: `internal/infrastructure/rdb/repository/`

## ディレクトリ構成

集約ごとに 1 つのディレクトリを置き、集約名を付ける。その集約の `.sql` を収める。

## 命名規則

- ファイル名: 動詞 + 対象名（例: `select_user_by_id.sql`）
- 全てのクエリに `-- name:` アノテーションが必須

## コード生成

```sh
make gen-query
```
