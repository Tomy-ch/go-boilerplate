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

## テスト方針

CLI 層は **driving adapter（humble object＝薄い殻）** です。担保戦略は意図的に分割しており、**その分割はカバレッジの数字に暗示させるのではなく、ここ（散文）に明記**します —— カバレッジは測定値であって、「取り扱い注意」を符号化する場所ではありません。

- **殻に silent-wrong なロジックを置かない。** 判断（エラー処理・分岐・整形・削除可否・タイムアウト分岐）は必ず純粋な関数/メソッドへ抽出し、**分岐網羅でユニットテスト**する。その周りの薄い `RunE`・配線はユニットテストしない。
- **OS / ファイルシステム / 外部プロセス / DB 依存は interface で注入する。** プロダクションコードは実装を結線し、ユニットテストはフェイクを渡す。よって**テストは実ファイルシステムに触れず、外部バイナリ（`pg_dump` / `psql`）を実行せず、DB も開かない**。
- **残りを担保するもの（ユニットではない）:**
  - DB アクセス挙動 → 実 Postgres に当てた repository テスト（`internal/infrastructure/rdb/testkit`）。
  - 「入口が実際に動くか」→ CI boot チェック: `app-di-startup-check`（serve → `/ready`）、`job-boot-check`（job dispatch）、`migration-check`（up/down 往復）、`gen-*-artifacts-check`（codegen の dogfooding）——いずれも実 Postgres サービス相手。
- **なぜ `cli` をユニットカバレッジから除外するか**（`makefile` の `TGT_PKGS`）: 殻は意図的に薄く、その実行時挙動は上記 CI ゲートで検証しているため。**低い/無いユニットカバレッジを「未テスト」と読まないこと** —— この節を読むこと。

### コマンドを追加するとき

- `RunE` は薄く保つ: フラグ解釈 → 依存の組み立て → テスト可能な関数へ委譲。
- フラグは**ローカル変数**に束縛する（パッケージグローバルにしない）。並列テスト安全のため。
- OS / FS / exec / DB は interface 経由で注入し、実装を既定値として渡す。
- 抽出したロジックを分岐網羅でユニットテストし、薄い殻は CI ゲートに委ねる。

## 注意点

- マイグレーションやシード操作は、実行前にバックアップを取ることを推奨
- サーバー起動時の設定は環境変数で管理（`internal/config` 参照）
- このディレクトリはインフラ層であり、AI エージェントは明示的な指示がない限り変更しないこと
