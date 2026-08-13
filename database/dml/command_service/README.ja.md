# CommandService DML

[English](README.md) | 日本語

Repository の load-mutate-save の形に収まらない状態変更（INSERT / UPDATE / DELETE）のための書き込み SQL を管理します。

## 目的

- リポジトリのCRUDや読み取り専用クエリとは分離して、状態変更を伴う書き込み操作を集約します。
- 複数テーブルの更新や複雑な状態遷移に対するトランザクション整合性を確保します。
- sqlc によるコード生成で、パラメータの型をコンパイル時に検証します。

## インフラストラクチャマッピング

実装: `internal/infrastructure/rdb/command_service/`

## ディレクトリ構成

集約または関心事ごとに 1 つのサブディレクトリを置き、その単位の `.sql` を収める。

- `purchase/` — 在庫の増減と原子的に行う必要のある購入の書き込み（作成・キャンセル）

## 命名規則

- ファイル名: 動詞 + 対象名（例: `update_user_email.sql`）
- 全てのクエリに `-- name:` アノテーションが必須

## コード生成

```sh
make gen-query
```
