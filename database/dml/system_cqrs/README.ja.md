# SystemQuery DML

[English](README.md) | 日本語

ヘルスチェック、メトリクス収集、インフラ監視のためのシステム運用クエリを管理します。

## 目的

- システムの健全性確認や運用メトリクスのためのクエリを提供します。
- システムレベルの関心事をアプリケーションのビジネスロジックから分離します。
- sqlc によるコード生成で、パラメータやスキャンの型をコンパイル時に検証します。

## インフラストラクチャマッピング

実装: `internal/infrastructure/rdb/system_cqrs/`

## ディレクトリ構成

```text
system_cqrs/
├── health_check/
│   ├── select_system_health.sql
│   └── ...
├── metrics/
│   ├── select_system_metrics.sql
│   └── ...
└── ...
```

## 命名規則

- ファイル名: 動詞 + 対象名（例: `select_system_health.sql`）
- 全てのクエリに `-- name:` アノテーションが必須

## コード生成

```sh
make gen-query
```
