---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [persistence, codegen, tooling]
---

# ADR-0028: マージされたDMLおよびダンプされたスキーマをsqlcの単一入力として使用する

English canonical: [0028-merged-dml-schema-as-sqlc-input.md](0028-merged-dml-schema-as-sqlc-input.md)

## ステータス

accepted

## 背景

sqlcには2種類の入力が必要である。スキーマ（テーブル定義）とSQLクエリファイルである。スキーマはライブデータベースに適用されるマイグレーションを通じて変化し、DMLクエリはカテゴリごとのサブディレクトリ（`repository/`・`query_service/`・`command_service/`・`system_cqrs/`）に分散している。sqlcをマイグレーションファイルに直接向けることは現実的でない。sqlcはマイグレーションの順序を理解しないため、すべてのDDL文を順序通りにマージして適用する必要があり、それはまさに`dump-schema`がライブDBから取り込む作業である。分散したDMLディレクトリをマージせずにsqlcに向けると、sqlcがディレクトリレイアウトを把握する必要が生じる。

既知の場所に単一の自己完結した入力セットを生成する前処理ステップを設けることで、sqlcの設定をシンプルに保ち生成パイプラインを決定論的にする。

## 決定

`sqlc generate`を実行する前に、`database/gen/`配下にsqlcの統一入力セットを生成する2つのビルドステップを設ける。

1. **merge-dml**（`go run ./cmd/ merge-dml --type=$(type) --work-dir=$(work-dir)`）が各DMLカテゴリディレクトリのすべてのSQLファイルを`database/gen/`に連結し、カテゴリごとに1つのマージファイルを生成する。
2. **dump-schema**（`go run ./cmd/ dump-schema --work-dir=$(work-dir)`）がライブのマイグレーション済みデータベースのスキーマを`database/gen/schema.gen.sql`にダンプする。

`sqlc.yaml`はこの2つの成果物のみを指定する。

```yaml
schema: database/gen/schema.gen.sql
queries: database/gen/
```

`make gen-query`はmerge-dml・dump-schema・sqlc generateの順でこれらのステップを実行する。`database/gen/`は生成された成果物であり、手動で編集してはならない。

## 影響

### ポジティブな影響

- sqlcは常に実際に適用されたマイグレーション状態を反映するスキーマを参照し、順序が乱れたり部分的に適用された生DDLファイルを参照しない。
- DMLファイルは`database/dml/`配下でカテゴリごとに整理されたまま生成前にマージされるため、人間の整理とツールのシンプルさの両方が保たれる。
- 生成されたGoコード（`internal/infrastructure/rdb/sqlc/gen/`）は同一のマイグレーション済みDB状態から完全に再現可能である。
- コミットされた`schema.gen.sql`のスナップショットにより、レビュアーはマイグレーションパイプラインを再実行せずに生成されたGoコードをレビューできる。生成されるGoファイルの実行時のSource of TruthはSQLファイルだが、生成時のスキーマ状態は生成実行時点のローカルDBにのみ存在するため、スナップショットとして保存することで再導出せずにレビューが可能になる。

### ネガティブな影響

- `make gen-query`を実行する前にライブのマイグレーション済みデータベースが利用可能でなければならない。スキーマダンプはマイグレーションファイルだけからは生成できない。通常開発時は常にローカルコンテナが起動しているため、実際には無害である。
- `database/gen/`はコミットするか、または生成時にDBコンテナへの依存を追加してCIで再生成しなければならない。

## 検討した代替案

### sqlcをマイグレーションファイルに直接向ける

sqlcはマイグレーションの順序を理解しない。すべてのマイグレーションファイルからすべてのDDL文を順序通りに適用する必要があり、それはまさに`dump-schema`がライブDBから取り込む作業と同一である。マイグレーションエンジンの仕事を複製するため却下。

### 手書きのschema.sqlを管理する

手動で管理されるスキーマファイルは実際に適用されたマイグレーションから時間とともにずれていく。dump-schemaアプローチは適用済み状態からスキーマを導出しドリフトを排除する。却下。

### DMLカテゴリごとに個別のsqlc呼び出し

各カテゴリ（`repository`・`query_service`等）ごとにカテゴリ別の設定でsqlcを個別に実行すると、別々の生成パッケージが生成されGoのインポートグラフが複雑になる。単一のマージされた入力により一貫性のある単一の生成パッケージが得られる。却下。

## 補足

- Source: [`database/README.md`](../../database/README.ja.md) — データライフサイクル図。
- 関連: [ADR-0026](0026-sql-first-data-access.ja.md)（SQLファーストデータアクセス）、[ADR-0027](0027-sqlc-type-safe-sql.ja.md)（コードジェネレーターとしてのsqlc）。
- `make gen-query`がmerge-dml・dump-schema・sqlcを順に実行する単一コマンドである。
