# Make コマンド一覧

[English](README.md) | 日本語

## 役割

`.makefiles/` はプロジェクトで使用するすべての `make` ターゲットの中央レジストリです。各 `.mk` ファイルは関連ターゲットを領域別（application / database / sql / go / openapi / docs / github / tools）にグルーピング。トップレベルの `makefile` はそれらを `include` するだけなので、新規ターゲット追加は該当グループファイルへの追記だけで完結し、トップレベル編集は不要です。

Make ターゲットは主に以下の単位で整理されています。

- `.makefiles/app` : アプリケーション起動・Job 実行
- `.makefiles/database` : DB 初期化 / マイグレーション / シード / DML / スキーマ
- `.makefiles/sql` : SQL Lint / Fix
- `.makefiles/markdown` : Markdown Lint / Fix
- `.makefiles/openapi` : OpenAPI バンドル / API ドキュメント生成
- `.makefiles/go` : Go コード生成 / フォーマット / Lint / テスト / ツール管理
- `.makefiles/docs` : Portal / ツール情報などのドキュメント生成
- `.makefiles/gen` : 各種生成処理の一括実行
- `.makefiles/github` : GitHub 初期設定 / リリース / ラベル / ルール設定

## 命名規約

- ターゲット名はハイフン区切りの小文字（`make new-migrate-<name>`、`make gen-api`）
- ターゲットは 2 種類:
  - **通常ターゲット**: 開発者がローカルで呼ぶ。再現性のため Docker コンテナ経由で実行
  - **`-ci` ターゲット**: CI ランナー、またはツールをローカルインストール済みの開発者向け低レベルコマンド
- すべて `.PHONY` 指定し、末尾 `##` コメントで `make help` 出力に載せること

## 補足

- 新規 `.mk` ファイルは該当グループ配下に追加すれば OK。`makefile` 側で既に `include` 済み
- ファイル直接作成より `make new-migrate-<name>` 等のヘルパーを優先（命名規約と番号採番を自動化）
- 一回限りの運用コマンド（`make setup-repo` 等）は `.makefiles/github/operation/` 配下に置き、開発者向けターゲットと分離する

## `.makefiles/app` 系

アプリケーションの開発環境起動や Job 実行に関するターゲット群です。

### アプリケーション起動関連

| コマンド | 説明 | 主な用途 |
| --- | --- | --- |
| `make serve` | `development` プロファイルの Docker Compose サービスをバックグラウンドで起動します。 | 通常のローカル開発開始 |
| `make serve-build` | Docker イメージをキャッシュを利用して再ビルドしたうえで開発環境を起動します。 | Dockerfile や依存変更の反映 |
| `make serve-build-clean` | `--no-cache --pull` でクリーンビルドしたうえで開発環境を起動します。 | base image 更新の取り込み（例: Go バージョンアップ） |
| `make tools` | `tools` プロファイルの開発支援ツール群を起動します。 | 開発ツール利用時 |
| `make tools-build` | 開発用ツールコンテナをキャッシュを利用してビルドします（起動はしません）。 | ツールコンテナの Dockerfile や依存変更の反映 |
| `make tools-build-clean` | 開発用ツールコンテナを `--no-cache --pull` 付きでクリーンビルドします（起動はしません）。 | ツールコンテナの base image 更新の取り込み |
| `make smoke` | `smoke` プロファイルの `smoke_server` をビルド付きで起動します。 | Smoke Test 環境の確認 |

#### `make job NAME=<job名> ARGS="<引数>"`

アプリケーションの Job を実行します。
`development` プロファイルのネットワーク内で `cmd/main.go job` を呼び出します。

- `NAME`: 実行する Job 名
- `ARGS`: Job に渡す追加引数（任意）

例:

```sh
make job NAME=sample-job
make job NAME=batch-import ARGS="--target=local --dry-run"
```

## `.makefiles/database` 系

DB 操作全般を扱うターゲット群です。
マイグレーション、シード投入、DML マージ、スキーマ生成、DB 初期化などを提供します。

### DB 初期化関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make db-init` | LocalDB と TestDB の初期化をまとめて実行します。 | `db-init-local` と `db-init-test` を順に呼び出します。 |
| `make db-init-local` | LocalDB を初期化します。 | `db-local-migrate-down` → `db-local-migrate-up` → `db-local-seed` を実行します。 |
| `make db-init-test` | TestDB を初期化します。 | `db-test-migrate-down` → `db-test-migrate-up` → `db-test-seed` を実行します。 |

### DB マイグレーション関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make new-migrate-<name>` | 新しいマイグレーションファイルを生成します。 | `database/migrations` 配下に連番付きの `.up.sql` / `.down.sql` を作成します。 |
| `make check-migration-up-version` | `up` 側マイグレーションのバージョン重複をチェックします。 | なし |
| `make check-migration-down-version` | `down` 側マイグレーションのバージョン重複をチェックします。 | なし |
| `make check-migration-up-gap` | `up` 側マイグレーションの連番ギャップをチェックします。 | なし |
| `make check-migration-down-gap` | `down` 側マイグレーションの連番ギャップをチェックします。 | なし |
| `make db-migrate-up DB=<database>` | 指定した DB に対して、全マイグレーションを最新まで適用します。 | 例: `make db-migrate-up DB=local` |
| `make db-migrate-up-<steps> DB=<database>` | 指定した DB に対して、現在位置から指定段数だけマイグレーションを適用します。 | 例: `make db-migrate-up-2 DB=local` |
| `make db-migrate-down DB=<database>` | 指定した DB に対して、全マイグレーションを初期状態までダウングレードします。 | なし |
| `make db-migrate-down-<steps> DB=<database>` | 指定した DB に対して、指定段数だけダウングレードします。 | なし |
| `make db-local-migrate-up` | LocalDB に対して全マイグレーションを最新まで適用します。 | `DB=local` を指定した `db-migrate-up` のエイリアスです。 |
| `make db-local-migrate-up-<steps>` | LocalDB に対して指定段数だけマイグレーションを適用します。 | なし |
| `make db-local-migrate-down` | LocalDB を初期状態までダウングレードします。 | `DB=local` を指定した `db-migrate-down` のエイリアスです。 |
| `make db-local-migrate-down-<steps>` | LocalDB を指定段数だけダウングレードします。 | なし |
| `make db-test-migrate-up` | TestDB に対して全マイグレーションを最新まで適用します。 | `DB=test` を指定した `db-migrate-up` のエイリアスです。 |
| `make db-test-migrate-up-<steps>` | TestDB に対して指定段数だけマイグレーションを適用します。 | なし |
| `make db-test-migrate-down` | TestDB を初期状態までダウングレードします。 | `DB=test` を指定した `db-migrate-down` のエイリアスです。 |
| `make db-test-migrate-down-<steps>` | TestDB を指定段数だけダウングレードします。 | なし |
| `make db-migrate-ci-up DB=<database>` | Docker を介さず、直接 `cmd/main.go migrate-up` を実行します。 | CI 用ターゲットです。 |
| `make db-migrate-ci-up-<steps> DB=<database>` | Docker を介さず、指定段数だけ `migrate-up` を実行します。 | CI 用ターゲットです。 |
| `make db-migrate-ci-down DB=<database>` | Docker を介さず、直接 `cmd/main.go migrate-down` を実行します。 | CI 用ターゲットです。 |
| `make db-migrate-ci-down-<steps> DB=<database>` | Docker を介さず、指定段数だけ `migrate-down` を実行します。 | CI 用ターゲットです。 |

例:

```sh
make new-migrate-create_users_table
make db-migrate-up DB=local
make db-migrate-up-10 DB=local
```

### DB シード関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make db-seed DB=<database>` | 指定した DB に対してシードデータを投入します。 | Docker コンテナ内で `cmd/main.go db-seed` を実行します。 |
| `make db-seed-ci DB=<database>` | Docker を介さず、直接シード投入処理を実行します。 | CI 用ターゲットです。 |
| `make db-local-seed` | LocalDB に対してシードデータを投入します。 | `DB=local` を指定した `db-seed` のエイリアスです。 |
| `make db-test-seed` | TestDB に対してシードデータを投入します。 | `DB=test` を指定した `db-seed` のエイリアスです。 |

### DB 生成・補助関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gen-db-schema` | DB スキーマドキュメントを生成します。 | ER 図やスキーマ出力の更新に使用します。 |
| `make gen-db-schema-ci` | SchemaSpy コンテナを直接実行してスキーマドキュメントを生成します。 | CI 用ターゲットです。 |
| `make dump-schema` | スキーマダンプを実行します。 | SQLC 生成や DML マージの前処理として利用します。 |
| `make dump-schema-ci` | Docker を介さず、直接 `cmd/main.go dump-schema` を実行します。 | CI 用ターゲットです。 |
| `make fix-collation` | データベースのコラテーションを修正します。 | なし |
| `make fix-collation-ci` | Docker を介さず、直接コラテーション修正処理を実行します。 | CI 用ターゲットです。 |

### DML マージ関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make merge-dml` | すべての DML マージ処理を実行します。 | `merge-dml-repo` → `merge-dml-qs` → `merge-dml-sysq` → `merge-dml-cs` を順に実行します。 |
| `make merge-dml-repo` | Repository 用 DML をマージします。 | なし |
| `make merge-dml-qs` | Query Service 用 DML をマージします。 | なし |
| `make merge-dml-cs` | Command Service 用 DML をマージします。 | なし |
| `make merge-dml-sysq` | System Query 用 DML をマージします。 | なし |
| `make merge-dml-core type="<type>" work-dir="<dir>"` | 指定した種別の DML マージを実行します。 | Docker コンテナ経由で `make merge-dml-ci-core` を呼び出します。 |
| `make merge-dml-ci` | すべての DML マージ処理を直接実行します。 | CI 用ターゲットです。 |
| `make merge-dml-ci-repo` | Repository 用 DML をマージします。 | CI 用ターゲットです。 |
| `make merge-dml-ci-qs` | Query Service 用 DML をマージします。 | CI 用ターゲットです。 |
| `make merge-dml-ci-cs` | Command Service 用 DML をマージします。 | CI 用ターゲットです。 |
| `make merge-dml-ci-sysq` | System Query 用 DML をマージします。 | CI 用ターゲットです。 |
| `make merge-dml-ci-core type="<type>" work-dir="<dir>"` | `cmd/main.go merge-dml` を直接実行します。 | CI 用ターゲットです。 |

例:

```sh
make merge-dml-core type="repository" work-dir="/app"
```

## `.makefiles/sql` 系

SQL ファイルに対する静的検査と自動修正を扱うターゲット群です。
対象は Migration / DML / Seed SQL です。

### SQL Lint 関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make sql-lint` | SQL の Lint を一括実行します。 | `sql-lint-migrations` → `sql-lint-dml` → `sql-lint-seed` を順に実行します。 |
| `make sql-lint-migrations` | マイグレーション SQL の Lint を実行します。 | なし |
| `make sql-lint-dml` | DML SQL の Lint を実行します。 | なし |
| `make sql-lint-seed` | シードデータ SQL の Lint を実行します。 | なし |
| `make sql-lint-migrations-ci` | `database/migrations/` に対して `sqlfluff lint` を実行します。 | CI 用ターゲットです。 |
| `make sql-lint-dml-ci` | `database/dml/` に対して `sqlfluff lint` を実行します。 | CI 用ターゲットです。 |
| `make sql-lint-seed-ci` | `database/seed/` に対して `sqlfluff lint` を実行します。 | CI 用ターゲットです。 |

### SQL Fix 関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make sql-fix` | SQL の自動修正を一括実行します。 | `sql-fix-migrations` → `sql-fix-dml` → `sql-fix-seed` を順に実行します。 |
| `make sql-fix-migrations` | マイグレーション SQL の自動修正を実行します。 | なし |
| `make sql-fix-dml` | DML SQL の自動修正を実行します。 | なし |
| `make sql-fix-seed` | シードデータ SQL の自動修正を実行します。 | なし |
| `make sql-fix-migrations-ci` | `database/migrations/` に対して `sqlfluff fix` を実行します。 | CI 用ターゲットです。 |
| `make sql-fix-dml-ci` | `database/dml/` に対して `sqlfluff fix` を実行します。 | CI 用ターゲットです。 |
| `make sql-fix-seed-ci` | `database/seed/` に対して `sqlfluff fix` を実行します。 | CI 用ターゲットです。 |

## `.makefiles/markdown` 系

Markdown ファイルに対する Lint と自動修正を扱うターゲット群です。

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make md-lint` | Markdown ファイルの Lint を実行します。 | `node_tool_runner` コンテナ内で `make md-lint-ci` を呼び出します。 |
| `make md-fix` | Markdown ファイルの Lint 自動修正を実行します。 | `node_tool_runner` コンテナ内で `make md-fix-ci` を呼び出します。 |
| `make md-lint-ci` | `markdownlint-cli2` で `**/*.md` を直接 Lint します。 | CI 用ターゲットです。`vendor/`、`node_modules/`、`.git/` を除外します。 |
| `make md-fix-ci` | `markdownlint-cli2 --fix` で `**/*.md` を直接修正します。 | CI 用ターゲットです。`vendor/`、`node_modules/`、`.git/` を除外します。 |

## `.makefiles/openapi` 系

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gen-bundle-oapi` | 分割された OpenAPI 定義をバンドルし、単一の OpenAPI ファイルを生成します。 | `openapi/openapi.yaml` をもとに `openapi/openapi.gen.yaml` を生成します。 |
| `make gen-api-docs` | OpenAPI 定義をもとに API ドキュメントを生成します。 | なし |
| `make gen-bundle-oapi-ci` | `redocly bundle` により `openapi/openapi.gen.yaml` を生成します。 | CI 用ターゲットです。 |
| `make gen-api-docs-ci` | `redocly build-docs` により `docs/openapi/index.html` を生成します。 | CI 用ターゲットです。 |

## `.makefiles/go` 系

### Go 生成関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gen-go-code` | Go のコード生成を実行します。 | Docker コンテナ内で `go generate ./...` を実行します。 |
| `make gen-go-code-ci` | Docker を介さず、`go generate ./...` を直接実行します。 | CI 用ターゲットです。 |
| `make gen-sqlc` | SQLC のコード生成をまとめて実行します。 | `remove-generated-sqlc` → `sqlc-generate` を順に実行します。 |
| `make remove-generated-sqlc` | 既存の SQLC 生成コードを削除します。 | なし |
| `make sqlc-generate` | SQLC によるコード生成を実行します。 | なし |
| `make remove-generated-sqlc-ci` | `$(SQLC_OUT)` 配下の `*.gen.sql.go` を削除します。 | CI 用ターゲットです。 |
| `make sqlc-generate-ci` | `sqlc generate -f sqlc.yaml` を直接実行します。 | CI 用ターゲットです。 |

### Go フォーマット・Lint・依存更新関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make fmt` | Go コードをフォーマットします。 | `go fmt ./...` を実行します。 |
| `make lint` | GolangCI-Lint による静的解析を実行します。 | なし |
| `make fix` | GolangCI-Lint の自動修正を実行します。 | なし |
| `make tidy-lib` | Go モジュール依存関係を整理し、`vendor` を更新します。 | `go mod tidy` と `go mod vendor` を順に実行します。 |

### Go テスト関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make test` | CI 用のテストを実行します。 | `gen` / `cli` / `cmd` / `mock` / `apperror` / `scripts` を除外したパッケージ群に対して `go test` を実行します。 |
| `make gen-test-repo` | テストを実行し、HTML カバレッジレポートを生成します。 | 出力先は `docs/coverage/index.html` です。 |
| `make test-cover-ci` | カバレッジ付きでテストを実行します。 | CI 用ターゲットで、`coverage.out` を出力します。 |

### Go ツールインストール関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make go-update` | `mise.toml` に記載された Go ランタイムを mise でインストールします。詳細は `docs/maintenance/go-upgrade.md` を参照。 | mise が必須 |
| `make install-tools` | host 開発用の Go ツール群を mise でインストールします（バージョンは `mise.toml` から解決）。 | `gopls`、`gotests`、`impl`、`dlv`、`lefthook`、`golangci-lint` を導入します。 |
| `make activate-tools` | `lefthook install` を実行し、Git フックをセットアップします。 | なし |

## `.makefiles/docs` 系

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gen-portal-docs` | Portal 用ドキュメントを生成します。 | なし |
| `make gen-docs-json` | Portal 用ドキュメントリンク JSON を生成します。 | なし |
| `make gen-portal-docs-ci` | Node.js スクリプトで Portal 用ドキュメントを直接生成します。 | CI 用ターゲットです。 |
| `make gen-docs-json-ci` | Node.js スクリプトで Portal 用 JSON を直接生成します。 | CI 用ターゲットです。 |

## `.makefiles/gen` 系

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gen` | 各種コード・ドキュメント生成をまとめて実行します。 | `gen-api` → `gen-query` → `gen-docs` を順に実行します。 |
| `make gen-api` | API 関連の生成処理をまとめて実行します。 | `gen-bundle-oapi` →  `gen-api-docs` → `gen-go-code` を実行します。 |
| `make gen-docs` | ドキュメント関連の生成処理をまとめて実行します。 | `gen-api-docs`、`gen-portal-docs`、`gen-docs-json` を実行します。 |
| `make gen-all-docs` | すべてのドキュメント生成処理を実行します。 | `gen-docs`、`gen-db-schema`、`gen-test-repo` を実行します。 |
| `make gen-query` | SQLC コード生成をまとめて実行します。 | `dump-schema` → `merge-dml` → `gen-sqlc` → `fmt` を順に実行します。 |
| `make gen-query-repo` | Repository 用 SQLC コード生成を実行します。 | `dump-schema` → `merge-dml-repo` → `gen-sqlc` を実行します。 |
| `make gen-query-qs` | Query Service 用 SQLC コード生成を実行します。 | `dump-schema` → `merge-dml-qs` → `gen-sqlc` を実行します。 |
| `make gen-query-sysq` | System Query 用 SQLC コード生成を実行します。 | `dump-schema` → `merge-dml-sysq` → `gen-sqlc` を実行します。 |

## `.makefiles/github` 系

### GitHub 設定関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gh-login` | `gh` コマンドで GitHub にログインします。 | ブラウザ認証方式でログインを行います。 |
| `make delete-all-labels` | GitHub リポジトリ上の既存ラベルをすべて削除します。 | なし |
| `make create-default-labels` | `.github/settings/labels.json` をもとに、デフォルトラベルを作成します。 | なし |
| `make apply-branch-protection` | `.github/settings/branch-protection.json` をもとに、対象リポジトリへブランチルールセットを適用します。 | なし |

### GitHub リポジトリ初期化関連

#### `make setup-repo`

リポジトリの初期化処理をまとめて実行します。
以下を順に行います。

- `gh` ログイン
- 初期タグ `v0.0.0` の作成と push
- `develop` / `staging` / `production` ブランチの作成
- GitHub デフォルトブランチの設定
- ブランチルールセット適用
- ラベル初期化

新規リポジトリを boilerplate として立ち上げる際の初期セットアップ用コマンドです。

#### セットアップ補助コマンド

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make setup-replace-module OLD_MODULE=<old> NEW_MODULE=<new>` | Go モジュール名を一括置換します。 | `node_tool_runner` を使用して `go.mod` や import パスを更新します。 |
| `make setup-replace-app-metadata APP_NAME=<name> OPENAPI_TITLE=<title> COPILOT_TITLE=<title>` | アプリケーション名や OpenAPI タイトルなどのメタデータを一括置換します。 | README や OpenAPI 定義などに反映されます。 |
| `make setup-replace-repository-reference REPOSITORY=<org/repo>` | リポジトリ参照（GitHub URL など）を一括置換します。 | README やドキュメント内のリンクを更新します。 |
| `make setup-replace-license-copyright COPYRIGHT_HOLDER=<name> [COPYRIGHT_YEAR=<year>]` | LICENSE の著作権表記を更新します。 | 年は省略可能です。 |
| `make setup-remove-debug-handlers` | Debug 用ハンドラ一式を削除します。 | 本番利用時の不要コード削除に使用します。 |

### リリースブランチ関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make hotfix-patch` | `production` から hotfix ブランチを作成し、GitHub のデフォルトブランチに設定します。 | 現在の最新タグを基準に patch を 1 つ進めます。 |
| `make branch-patch` | `production` から patch リリース用ブランチを作成し、デフォルトブランチに設定します。 | 現在の最新タグを基準に patch バージョンを進めます。 |
| `make branch-minor` | `production` から minor リリース用ブランチを作成し、デフォルトブランチに設定します。 | 現在の最新タグを基準に minor バージョンを進めます。 |
| `make branch-major` | `production` から major リリース用ブランチを作成し、デフォルトブランチに設定します。 | 現在の最新タグを基準に major バージョンを進めます。 |

### リリースタグ関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make tag-patch` | patch バージョンを 1 つ進めたタグを作成し、GitHub Release を作成します。 | 現在の最新タグを基準とし、リリースノートには `.github/release/<version>.md` を使用します。 |
| `make tag-minor` | minor バージョンを進めたタグを作成し、GitHub Release を作成します。 | 現在の最新タグを基準にします。 |
| `make tag-major` | major バージョンを進めたタグを作成し、GitHub Release を作成します。 | 現在の最新タグを基準にします。 |
