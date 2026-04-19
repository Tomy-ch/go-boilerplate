# CLI

[English](README.md) | 日本語

`internal/cli` は、Cobra ベースの **コマンドラインインターフェース**を提供するパッケージです。

サーバー起動、データベースマイグレーション、シード投入、ジョブ実行など、アプリケーションの運用に必要なコマンドを定義しています。

## コマンド一覧

|コマンド|パッケージ|説明|
|---|---|---|
|`serve`|`server/`|HTTP サーバーとメトリクスサーバーを起動|
|`migrate-up`|`migrate/`|DDL をアップグレード（`--version` / `--database` 指定可）|
|`migrate-down`|`migrate/`|DDL をダウングレード（`--version` / `--database` 指定可）|
|`db-seed`|`seed/`|データベースに初期データを投入|
|`job`|`job/`|登録済みジョブを実行（`job <job-name> [args...]`）|
|`fix-collation`|`fixcollation/`|PostgreSQL の照合順序バージョン不一致を修正|
|`dump-schema`|`dumpschema/`|DB スキーマをダンプして整形|
|`merge-dml`|`mergedml/`|DML ディレクトリの SQL ファイルを種別ごとにマージ|

## 構造

```text
internal/cli/
├── cli.go              # RegisterCommands（サブコマンド登録）
├── server/             # serve コマンド + メトリクスサーバー
├── migrate/            # migrate-up / migrate-down
├── seed/               # db-seed
├── job/                # job <name>
├── fixcollation/       # fix-collation
├── dumpschema/         # dump-schema
└── mergedml/           # merge-dml
```

`cli.go` の `RegisterCommands` で全サブコマンドを Cobra のルートコマンドに登録します。

## 設計方針

- 各コマンドは1パッケージ = 1コマンドで分離
- CLI 層はビジネスロジックを持たない（DI 経由で Usecase を呼び出す）
- コマンドの追加は `cli.go` の `RegisterCommands` に追加するだけ

## 注意点

- マイグレーションやシード操作は、実行前にバックアップを取ることを推奨
- サーバー起動時の設定は環境変数で管理（`internal/config` 参照）
- このディレクトリはインフラ層であり、AI エージェントは明示的な指示がない限り変更しないこと
