# Repository DML

[English](README.md) | 日本語

ドメインエンティティの永続化（CRUD）に使用するSQLを管理します。

## 目的

- ドメインモデルの永続化・取得に必要な INSERT / UPDATE / DELETE / SELECT クエリを集約します。
- ビジネスロジックを含まないシンプルなSQLを維持します。集計や複雑な結合は QueryService で行います。
- sqlc によるコード生成で、型安全なリポジトリ実装を提供します。

## インフラストラクチャマッピング

実装: `internal/infrastructure/rdb/repository/`

## ディレクトリ構成

```text
repository/
├── user/
│   ├── insert_user.sql
│   ├── select_user_by_id.sql
│   └── ...
├── prefecture/
│   ├── ...
│   └── ...
└── ...
```

## 命名規則

- ファイル名: 動詞 + 対象名（例: `select_user_by_id.sql`）
- 全てのクエリに `-- name:` アノテーションが必須

## コード生成

```sh
make gen-query
```
