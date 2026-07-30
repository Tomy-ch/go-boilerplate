# Make コマンド一覧

[English](README.md) | 日本語

## 役割

`.makefiles/` はプロジェクトで使用するすべての `make` ターゲットの中央レジストリです。各 `.mk` ファイルは関連ターゲットを領域別（application / database / sql / go / openapi / docs / github / tools）にグルーピング。トップレベルの `makefile` はそれらを `include` するだけなので、新規ターゲット追加は該当グループファイルへの追記だけで完結し、トップレベル編集は不要です。

Make ターゲットは主に以下の単位で整理されています。

- `.makefiles/app` : アプリケーション起動・常駐プロセス(worker / outbox-relay)・Job 実行・埋め込み env 材料化
- `.makefiles/database` : DB 初期化 / マイグレーション / シード / DML / スキーマ
- `.makefiles/sql` : SQL Lint / Fix
- `.makefiles/markdown` : Markdown Lint / Fix
- `.makefiles/security` : Trivy 依存脆弱性スキャン
- `.makefiles/docker` : compose プロジェクト / ホストポート定義・Dockerfile Lint（hadolint）・image digest 固定
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

- 既存グループファイルへのターゲット追加ならトップレベル編集は不要。ただし新規 `.mk` ファイルを追加する場合は、トップレベル `makefile` へ `include` 行の追記が必要（ワイルドカードではなく個別 include のため）
- ファイル直接作成より `make new-migrate-<name>` 等のヘルパーを優先（命名規約と番号採番を自動化）
- 一回限りの運用コマンド（`make setup-repo` 等）は `.makefiles/github/operation/` 配下に置き、開発者向けターゲットと分離する

## `.makefiles/app` 系

アプリケーションの開発環境起動や Job 実行に関するターゲット群です。

compose のサービスは 2 層に分かれます（後述の `.makefiles/docker` 系を参照）。共有の **infra 層**
（`database` / `observability` / `garage`）は固定プロジェクト `gobp-shared` に 1 インスタンスだけ置き、
checkout 毎の **app 層**（`api_server` / `mock_auth_server`）は自 checkout の `APP_PROJECT` で起動します。

### アプリケーション起動関連

| コマンド | 説明 | 主な用途 |
| --- | --- | --- |
| `make serve` | 共有インフラを起動（`infra-up`）したうえで、自 checkout の app サービスをバックグラウンド起動し、DB スロットの heartbeat を更新します。 | 通常のローカル開発開始 |
| `make serve-build` | app イメージをキャッシュ利用で再ビルドし、共有インフラを起動したうえで app サービスを起動します。 | Dockerfile や依存変更の反映 |
| `make serve-build-clean` | app イメージを `--no-cache --pull` でクリーンビルドし、共有インフラを起動したうえで app サービスを起動します。 | base image 更新の取り込み（例: Go バージョンアップ） |
| `make serve-stop` | 自 checkout の app プロジェクトだけを停止します。 | 共有インフラや他 checkout に触れず API を止める |
| `make infra-up` | 共有インフラのサービス（`--wait`）と one-shot の `garage_init` を `gobp-shared` プロジェクトで起動します。 | 共有インフラだけを起動する（`serve` / `job` / `worker` が冪等に呼びます） |
| `make infra-down` | 共有インフラのプロジェクトを停止します（名前付きボリュームは保持）。 | インフラを落とす。**全 checkout / worktree に影響します** |
| `make tools` | `tools` プロファイルの開発支援ツール群を共有インフラのプロジェクトで起動します。 | 開発ツール利用時（SQL editor `:7000` / docs viewer `:7001`） |
| `make all` | `tools` → `serve-build` の順に全サービスを一括起動します。 | ローカルスタック全体を一度に立ち上げる |
| `make tool-runners-build` | オンデマンド実行のツールランナー画像(go/node/python)をキャッシュ利用でビルドします（起動はしません）。 | ツールランナーの Dockerfile や依存変更の反映 |
| `make tool-runners-build-clean` | ツールランナー画像を `--no-cache --pull` 付きでクリーンビルドします（起動はしません）。 | ツールランナーの base image 更新の取り込み |

#### `make job NAME=<job名> ARGS="<引数>"`

アプリケーションの Job を実行します。
共有インフラを起動したうえで、自 checkout の app プロジェクトの使い捨て `api_server` コンテナ
（`run --rm`）で `cmd/main.go job` を呼び出します。

- `NAME`: 実行する Job 名
- `ARGS`: Job に渡す追加引数（任意）

例:

```sh
make job NAME=sample-job
make job NAME=batch-import ARGS="--target=local --dry-run"
```

### 常駐プロセス(worker / outbox-relay)関連

いずれも `SIGTERM` / `Ctrl-C` まで常駐するデーモンで、共有インフラを起動したうえで自 checkout の
app プロジェクトの使い捨て `api_server` コンテナ内（`make job` と同じ `go run ./cmd/` 方式）で
実行します。

#### `make worker NAME=<worker名> ARGS="<引数>"`

pull-ack worker を起動します。`NAME` は worker 名（必須）、`ARGS` は任意です。

> スキャフォールドは既定で worker を1つも登録しません（`WorkerModule()` は空の seam）。
> そのため実 worker を配線するまでは `unknown worker` で失敗します。worker 追加後の
> ローカル動作確認用の起動口として置いています。

```sh
make worker NAME=sampleworker
```

#### `make outbox-relay ARGS="<引数>"`

outbox relay を起動します（outbox テーブルを周期 poll して未 publish メッセージを送出）。
`ARGS` は任意で、`replay` サブコマンドにも渡ります。

```sh
make outbox-relay
make outbox-relay ARGS="replay --message-id=<id>"
```

### 埋め込み env 材料化関連

サーバーバイナリは `env/.env` を埋め込みます。CI および Docker ビルドはビルド前に
環境別ファイルを `env/.env` へ材料化するため、その手順（と、ドリフト判定向けの取り消し）を
これらのターゲットへ集約します。

| コマンド | 説明 | 主な用途 |
| --- | --- | --- |
| `make materialize-env` | `env/.env.$(APP_ENV)` を `env/.env` へコピーします（既定は `APP_ENV=ci`）。 | CI / ビルドで `go build` / `go run` 前に埋め込み対象を材料化する |
| `make restore-env` | `git restore` で `env/.env` を git 管理の内容へ戻します。 | 生成物ドリフト / コミット判定の前に材料化を取り消す |

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
| `make md-lint` | Markdown の Lint を実行します（markdownlint + mermaid 構文 + スキル定義の意味検査）。 | `node_tool_runner` コンテナ内で `make md-lint-ci` を呼び出します。 |
| `make md-fix` | Markdown ファイルの Lint 自動修正を実行します。 | `node_tool_runner` コンテナ内で `make md-fix-ci` を呼び出します。 |
| `make md-mermaid-lint` | ` ```mermaid ` フェンスのみを構文検証します。 | `node_tool_runner` コンテナ内で `make md-mermaid-lint-ci` を呼び出します。 |
| `make md-skill-lint` | `.claude/**` のスキル / エージェント定義と、その `.codex/**` 対応のみを検証します。 | `node_tool_runner` コンテナ内で `make md-skill-lint-ci` を呼び出します。 |
| `make md-lint-ci` | `markdownlint-cli2` を実行後、mermaid 構文 Lint、スキル定義 Lint の順に実行します。 | CI 用ターゲットです。`vendor/`、`node_modules/`、`.git/` を除外します。 |
| `make md-mermaid-lint-ci` | `scripts/mermaid-lint.mjs`（実 `mermaid.parse`）で ` ```mermaid ` フェンスを検証します。 | CI 用ターゲット。markdownlint は図の文法を見ません。 |
| `make md-skill-lint-ci` | `scripts/skill-lint.mjs` で `.claude/**` の定義（frontmatter / 対訳ペアの構造 / 参照の実在性）と、`.codex/**` との対応（skill / agent の存在対応、Codex skill の構造）を検証します。 | CI 用ターゲット。markdownlint は記述と実態の一致を見ず、片側の環境にだけ入った skill も他の誰も気づきません。 |
| `make md-fix-ci` | `markdownlint-cli2 --fix` で `**/*.md` を直接修正します。 | CI 用ターゲットです。`vendor/`、`node_modules/`、`.git/` を除外します。 |

## `.makefiles/security` 系

CI のセキュリティ指摘をローカルで再現するためのスキャン（Trivy 依存スキャン、gitleaks シークレットスキャン）です。image スキャンは CI 専用（`image-scan.yaml`）です。

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make trivy-fs` | ライブラリ依存を Trivy fs でスキャンします。 | `go_tool_runner` コンテナ内で `make trivy-fs-ci` を呼び出します。 |
| `make trivy-fs-ci` | `trivy fs` を直接実行します。 | CI 用ターゲット。CI と揃えるため `vendor/` を除外します。 |
| `make trivy-fs-release-ci` | 修正版のない脆弱性も含めて `trivy fs` を実行します。 | 昇格ゲート用の CI ターゲット。`trivy-fs-ci` との差は `--ignore-unfixed` の有無だけです。 |
| `make trivy-config` | Dockerfile の設定不備をスキャンします。 | `go_tool_runner` コンテナ内で `make trivy-config-ci` を呼び出します。 |
| `make trivy-config-ci` | `trivy config` を直接実行します。 | CI 用ターゲット。`CRITICAL,HIGH` でゲートし、許容する例外は `.trivyignore.yaml` に置きます。 |
| `make trivy-license` | 依存ライブラリのライセンスを列挙します。 | `go_tool_runner` コンテナ内で `make trivy-license-ci` を呼び出します。 |
| `make trivy-license-ci` | `trivy fs --scanners license` を直接実行します。 | CI 用ターゲット。禁止ライセンス方針が未策定のため報告専用で、severity では絞りません。 |
| `make trivy-image-ci` | ビルド済みイメージの脆弱性をスキャンします。 | CI 用ターゲット。対象イメージは `TRIVY_IMAGE=` で渡します。 |
| `make trivy-image-gate-ci` | ビルド済みイメージの修正版のある `CRITICAL` / `HIGH` で失敗します。 | CI 用ターゲット。対象イメージは `TRIVY_IMAGE=` で渡します。 |
| `make secret-scan` | ワーキングツリーのシークレットを gitleaks でスキャンします。 | `go_tool_runner` コンテナ内で `make secret-scan-ci` を呼び出します。 |
| `make secret-scan-ci` | `gitleaks dir . --redact` を直接実行します。 | CI 用ターゲット。生成ファイルは `.gitleaks.toml` で allowlist。 |
| `make secret-scan-history-ci` | `gitleaks git . --redact` を直接実行します。 | CI 用ターゲット。週次実行が使用。`dir` は作業ツリーしか見ないためコミット後に消したシークレットを取りこぼすが、`git` は履歴全体を走査する。 |
| `make npm-cooldown-audit` | lockfile のエントリのうち、同階層 `.npmrc` の `min-release-age` を満たさないものを報告します。 | ホスト上で実行。報告のみで、検出があっても 0 で終了します（cooldown の解除は意図的な判断であるため）。 |

## `.makefiles/docker` 系

全ターゲットが共有する compose プロジェクト / ホストポートの定義を持ち、`go_tool_runner` コンテナ経由で
hadolint により Dockerfile を lint し、`FROM` の base image を不変の digest へ固定します
（サプライチェーン対策）。

### compose プロジェクト定義（`compose.mk`）

`compose.mk` はターゲットを持たず、app / database 系が土台にする変数を定義するため、トップレベル
`makefile` の冒頭（「依存されるファイル」セクション）で `include` されます。DB スロット保持時は
`.gobp-db-slot` が既定値を上書きします（`internal/cli/dbslot/README.ja.md` 参照）。

| 変数 | 既定 | 説明 |
| --- | --- | --- |
| `INFRA_PROJECT` | `gobp-shared` | 共有インフラの唯一のインスタンスを置く固定 compose プロジェクト。 |
| `APP_PROJECT` | `gobp-app-$(notdir $(CURDIR))` | app 層の checkout 毎 compose プロジェクト。DB スロット保持時は `SERVE_PROJECT`（`gobp-wt-N`）になります。 |
| `INFRA_SERVICES` | `database observability garage` | 固定ポートでしか動けないため共有するサービス。 |
| `APP_SERVICES` | `api_server mock_auth_server` | checkout 毎に起動するサービス。 |
| `COMPOSE_INFRA` | `docker compose -p $(INFRA_PROJECT)` | infra 層向けの compose 呼び出し。 |
| `COMPOSE_APP` | `docker compose -p $(APP_PROJECT) -f docker-compose.yaml -f docker-compose.attach.yaml --profile development` | app 層向けの compose 呼び出し。`docker-compose.attach.yaml` が app サービスの接続先を `host.docker.internal` 経由の共有インフラへ差し替えます。 |
| `API_HOST_PORT` / `MOCK_AUTH_HOST_PORT` | `8080` / `4000` | API / mock 認証サーバーのホスト公開ポート。 |
| `DLV_HOST_PORT` / `PPROF_HOST_PORT` | `2345` / `6060` | dlv デバッグ / pprof のホスト公開ポート。 |
| `COMPOSE_PROJECT_NAME` | `$(INFRA_PROJECT)` | `-p` を渡さない compose 呼び出しの既定プロジェクト。DB ツーリングが共有インフラのネットワークで動くようにします。 |

### Dockerfile Lint / image 固定関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make docker-lint` | `docker/*/Dockerfile` を hadolint で lint します。 | `go_tool_runner` コンテナ内で `make docker-lint-ci` を呼び出します。 |
| `make docker-lint-ci` | `hadolint docker/*/Dockerfile` を直接実行します。 | CI 用ターゲット。無効化ルールは `.hadolint.yaml`。 |
| `make pin-images-resolve` | `docker/*/Dockerfile` の `FROM` と `docker-compose*.yaml` の `image:` の `image:tag` を現在の digest へ解決し `docker/images-pin.toml` lockfile を更新します。 | `PIN_IMAGES_MIN_AGE_DAYS`（既定 14；0 で無効）日未満の digest は quarantine。registry アクセス（`docker`）が必要。 |
| `make pin-images-apply` | lockfile を元に `FROM` / compose `image:` を `image:tag@sha256:...` へ固定します（quarantine 中の image は tag のまま）。 | なし |
| `make pin-images-check` | `FROM` / compose `image:` が lockfile 通り固定済みか検証します（書き換えなし）。 | CI / pre-commit gate。 |

## `.makefiles/openapi` 系

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gen-bundle-oapi` | 分割された OpenAPI 定義をバンドルし、単一の OpenAPI ファイルを生成します。 | `openapi/openapi.yaml` をもとに `openapi/openapi.gen.yaml` を生成します。 |
| `make gen-api-docs` | OpenAPI 定義をもとに API ドキュメントを生成します。 | なし |
| `make lint-oapi` | OpenAPI 定義を `redocly lint` で検証します。 | `node_tool_runner` コンテナ内で `make lint-oapi-ci` を呼び出します。 |
| `make gen-bundle-oapi-ci` | `redocly bundle` により `openapi/openapi.gen.yaml` を生成します。 | CI 用ターゲットです。 |
| `make gen-api-docs-ci` | `redocly build-docs` により `docs/openapi/index.html` を生成します。 | CI 用ターゲットです。 |
| `make lint-oapi-ci` | `redocly lint openapi/openapi.yaml` を直接実行します。 | CI 用ターゲットです。 |
| `make lint-oapi-security-ci` | Spectral + OWASP API Security ルールセットで検証します。 | CI 用ターゲット。Spectral が ruleset を `docker/tools/node_modules` から解決するためコンテナを介さず実行します。事前に同ディレクトリで `npm ci` が必要です。 |
| `make gen-mock-auth-oapi` | mock-auth-server の OpenAPI をバンドルし zod スキーマを生成します。 | `node_tool_runner` コンテナ内で `make gen-mock-auth-oapi-ci` を呼び出します。 |
| `make gen-mock-auth-oapi-docs` | mock-auth-server の OpenAPI から Redoc HTML を生成します。 | `node_tool_runner` コンテナ経由で `docs/openapi/mock-auth-server/index.html` を出力します。 |
| `make lint-mock-auth-oapi` | mock-auth-server の OpenAPI 定義を `redocly lint` で検証します。 | `node_tool_runner` コンテナ内で `make lint-mock-auth-oapi-ci` を呼び出します。 |
| `make gen-mock-auth-oapi-ci` | `docker/mock-auth-server` で `npm run gen`（redocly bundle + orval）を実行します。 | CI 用ターゲットです。 |
| `make gen-mock-auth-oapi-docs-ci` | `docker/mock-auth-server` で `npm run gen:docs`（redocly build-docs）を実行します。 | CI 用ターゲットです。 |
| `make lint-mock-auth-oapi-ci` | `docker/mock-auth-server` で `npm run lint:oapi` を実行します。 | CI 用ターゲットです。 |

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
| `make test` | CI 用のテストを実行します。 | `gen` / `cmd` / `mock` / `apperror` / `scripts` を除外したパッケージ群に対して `go test` を実行します（`internal/cli` コアは計測対象に含まれます）。 |
| `make test-cached` | ローカル用にテストキャッシュを有効にしてテストを実行します。 | pre-commit のローカル実行向け。除外パッケージは `test` と同じですが、`-count=1` を付けずキャッシュ結果を再利用します。 |
| `make gen-test-repo` | テストを実行し、HTML カバレッジレポートを生成します。 | 出力先は `docs/coverage/index.html` です。 |
| `make test-cover-ci` | カバレッジ付きでテストを実行します。 | CI 用ターゲットで、`coverage.out` を出力します。 |
| `make cover-gate` | 総カバレッジが閾値を下回ると fail します。 | CI ゲート。`COVERAGE_THRESHOLD`（既定 90）。`coverage.out` が必要（先に `test-cover-ci`）。 |

### Go ツールインストール関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make go-update` | `mise.toml` に記載された Go ランタイムを mise でインストールします。詳細は `docs/maintenance/go-upgrade.md` を参照。 | mise が必須 |
| `make install-tools` | host 開発用の Go ツール群を mise でインストールします（バージョンは `mise.toml` から解決）。 | `gopls`、`gotests`、`impl`、`dlv`、`lefthook`、`golangci-lint` を導入します。 |
| `make activate-tools` | `lefthook install` を実行し、Git フックをセットアップします。 | なし |
| `make sync-versions` | `mise.toml` の go / node / python バージョンを `go.mod` と Dockerfile の `FROM` に反映します。 | `docs/maintenance/go-upgrade.md` の手順で参照されます。`scripts/sync-versions` を実行します。 |

## `.makefiles/docs` 系

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gen-portal-docs` | Portal 用ドキュメントを生成します。 | なし |
| `make gen-docs-json` | Portal 用ドキュメントリンク JSON を生成します。 | なし |
| `make gen-portal-build` | Portal フロントエンド（`docs/portal/src/main.jsx`）を esbuild で `bundle.js` / `bundle.css` にバンドルします。 | なし |
| `make gen-portal-docs-ci` | Node.js スクリプトで Portal 用ドキュメントを直接生成します。 | CI 用ターゲットです。 |
| `make gen-docs-json-ci` | Node.js スクリプトで Portal 用 JSON を直接生成します。 | CI 用ターゲットです。 |
| `make gen-portal-build-ci` | esbuild を直接実行して Portal フロントエンドをバンドルします。 | CI 用ターゲットです。 |
| `make gen-godoc` | godoc の静的 HTML を `docs/godoc/` に生成します。 | なし |
| `make gen-godoc-ci` | godoc-static を直接実行して静的 HTML を生成します。 | CI 用ターゲットです。 |

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

### GitHub Actions lint / pin 関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make actions-lint` | ワークフロー定義を actionlint で lint し、全 composite action の `run:` スクリプトを shellcheck で検査したうえで、node による 3 つの検査（PR コメントを投稿するジョブへの secret 混入、コメント本文の固定長フェンス、ジョブ打ち切り時の振る舞いが定義されているか）を実行します。 | 各段が 2 つの tool-runner に跨る唯一の lint グループ。actionlint と shellcheck ランナーは Go ツール、残りは node スクリプトのため、単一コンテナ内で 1 つの `-ci` を呼ぶのではなく `go_tool_runner` で `make actions-actionlint-ci` と `make actions-shellcheck-ci`、`node_tool_runner` で 3 つの `*-lint-ci` を呼び出します。 |
| `make actions-comment-secret-lint` | PR コメント本文への secret 混入検査のみを実行します。 | `node_tool_runner` コンテナ内で `make actions-comment-secret-lint-ci` を呼び出します。 |
| `make actions-comment-fence-lint` | PR コメント本文の固定長フェンス検査のみを実行します。 | `node_tool_runner` コンテナ内で `make actions-comment-fence-lint-ci` を呼び出します。 |
| `make actions-cutoff-lint` | ジョブ打ち切り時の振る舞い検査のみを実行します。 | `node_tool_runner` コンテナ内で `make actions-cutoff-lint-ci` を呼び出します。 |
| `make actions-shellcheck` | `.github/actions/**` の composite action から `runs.steps[].run` を抽出し、`bash` / `sh` のスクリプトを `shellcheck` で検査します。それ以外の shell のステップは skip として報告します（`scripts/actions-shellcheck`）。 | `go_tool_runner` コンテナ内で `make actions-shellcheck-ci` を呼び出します。`actionlint` は `.github/workflows` しか走査せず、`action.yaml` を直接渡すとワークフローとして解釈するため、その死角を埋めます。 |
| `make actions-lint-ci` | 上記 5 段を直接実行します。 | CI 用ターゲット。 |
| `make actions-actionlint-ci` | `actionlint` を直接実行します。 | CI 用ターゲット。 |
| `make actions-shellcheck-ci` | `scripts/actions-shellcheck` を直接実行します。 | CI 用ターゲット。 |
| `make actions-comment-secret-lint-ci` | `upsert-pr-comment` を使うジョブに `GITHUB_TOKEN` 以外の secret が渡っていれば失敗します（`scripts/pr-comment-secret-lint.mjs`）。 | CI 用ターゲット。規約の理由は [`.github/workflows/README.ja.md`](../.github/workflows/README.ja.md) を参照。 |
| `make actions-comment-fence-lint-ci` | `run:` ブロックが PR コメント本文を固定長 Markdown フェンスで囲んでいる場合、または複製された `fence_for` の実装が食い違う場合に失敗します（`scripts/pr-comment-fence-lint.mjs`）。 | CI 用ターゲット。規約の理由は [`.github/workflows/README.ja.md`](../.github/workflows/README.ja.md) を参照。 |
| `make actions-cutoff-lint-ci` | ジョブに `timeout-minutes` が無い場合、または `upsert-pr-comment` を呼ぶステップの `if:` がキャンセルされたジョブから到達できない場合に失敗します（`scripts/actions-cutoff-lint.mjs`）。 | CI 用ターゲット。規約の理由は [`.github/workflows/README.ja.md`](../.github/workflows/README.ja.md) を参照。 |
| `make pin-actions-resolve` | 各 `uses:` のタグを commit SHA に解決し `.github/actions-pin.toml` lockfile を更新します。 | `PIN_ACTIONS_MIN_AGE_DAYS`（既定 14・0 で無効）より新しい解決先を quarantine。 |
| `make pin-actions-apply` | lockfile を元に `uses:` を `@<sha> # <tag>` へ固定します。 | なし |
| `make pin-actions-check` | `uses:` が lockfile 通り固定済みか検証します（書き換えなし）。 | CI / pre-commit ゲート。 |

### コミットメッセージ Lint 関連

| コマンド | 説明 | 備考 |
| --- | --- | --- |
| `make commitlint COMMIT_MSG_FILE=<file>` | コミットメッセージを commitlint で検証します。 | `node_tool_runner` コンテナ内で `make commitlint-ci` を呼び出します。`commit-msg` フックに配線。`git worktree` では git がフックへ渡すパスがコンテナのマウント範囲 `.:/app` の外にあるため、メッセージファイルを `tmp/` へ写して相対パスで渡します。`COMMIT_MSG_FILE` 既定は `git rev-parse --git-path COMMIT_EDITMSG`。 |
| `make commitlint-ci COMMIT_MSG_FILE=<file>` | `commitlint --edit <file>` を直接実行します。 | CI 用ターゲット。 |

### GitHub 設定関連

| コマンド | 説明 | 補足 |
| --- | --- | --- |
| `make gh-login` | `gh` コマンドで GitHub にログインします。 | ブラウザ認証方式でログインを行います。 |
| `make delete-all-labels` | GitHub リポジトリ上の既存ラベルをすべて削除します。 | なし |
| `make create-default-labels` | `.github/settings/labels.json` をもとに、デフォルトラベルを作成します。 | なし |
| `make apply-branch-protection` | `.github/settings/branch-protection.json` をもとに、対象リポジトリへブランチルールセットを適用します。 | なし |
| `make enable-workflows` | `disabled_fork` 状態のワークフローを一括で有効化します。 | 冪等です。fork / テンプレート由来のリポジトリは全ワークフローが無効の状態で作られます。 |

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
| `make setup-replace-codeowners OWNERS='<owners>'` | `.github/CODEOWNERS` の全ルールの所有者を一括置換します。 | `@user` / `@org/team` / メールアドレスを指定でき、空白区切りで複数指定できます。コメント行は対象外なので、ヘッダーの記載例は書き換わりません。 |
| `make setup-remove-sample-api` | サンプルAPI(`user`/`product`/`order`)を一括削除します。 | `node_tool_runner` で削除後、`gen-api` → `gen-query` → `fix` → `lint` を実行します。**DB コンテナ(`database`)の起動が必要**（`gen-query` がライブスキーマをダンプ）。削除後は `make db-init-local db-init-test && make gen-query` で再構築し、削除済みテーブルが生成モデルに残らないようにします。`DRY_RUN=1` で変更せずプレビューできます（`0` を含む空でない値はすべてプレビュー扱いになるため、実行時は変数自体を付けません）。 <!-- sample-api:line --> |

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
