# dump-schema

[English](README.md) | 日本語

`pg_dump` を使用してデータベーススキーマをダンプし、ツール固有のメタコマンドを除去して sqlc で読み込める形式に整形します。

## コマンド

```
dump-schema [flags]
```

## フラグ

| フラグ | デフォルト | 説明 |
|--------|------------|------|
| `--work-dir` | `/app` | 作業ディレクトリ（プロジェクトルート） |

## 使い方

```bash
./server dump-schema

./server dump-schema --work-dir /app
```

## 注意点

- 出力先は `database/gen/schema.gen.sql` です。
- `pg_dump` が `PATH` 上に存在し、アプリケーションの DSN でデータベースに接続できる必要があります。
- `\` で始まる行（psql メタコマンド）は出力から自動的に除去されます。
- デフォルトの `pg_dump` フラグ: `--schema-only --no-owner --no-privileges --format=plain`。
