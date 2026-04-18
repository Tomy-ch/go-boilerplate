# CommandService DML

[English](README.md) | 日本語

状態変更（INSERT / UPDATE / DELETE）のための書き込みSQLを管理します。将来の拡張ポイントとして設計されています。

## 目的

- リポジトリのCRUDや読み取り専用クエリとは分離して、状態変更を伴う書き込み操作を集約します。
- 複数テーブルの更新や複雑な状態遷移に対するトランザクション整合性を確保します。
- sqlc によるコード生成で、パラメータの型をコンパイル時に検証します。

## インフラストラクチャマッピング

実装: `internal/infrastructure/rdb/command_service/`（将来）

## ディレクトリ構成

```text
command_service/
├── user/
│   ├── insert_user.sql
│   ├── update_user_email.sql
│   └── ...
├── product/
│   ├── publish_product.sql
│   └── ...
└── ...
```

## 命名規則

- ファイル名: 動詞 + 対象名（例: `update_user_email.sql`）
- 全てのクエリに `-- name:` アノテーションが必須

## コード生成

```sh
make gen-query
```
