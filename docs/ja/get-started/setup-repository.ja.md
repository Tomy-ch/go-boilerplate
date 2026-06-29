# リポジトリ複製後の作業リスト

[English](../../get-started/setup-repository.md) | 日本語

Makeコマンド詳細は [Makeターゲット一覧](.makefiles/README.ja.md) を参照してください。

## Phase 1: ツールのセットアップ

VSCode開発で必要なツールをインストールします。

### 1.1. mise のインストールとシェル activate

このプロジェクトは [mise](https://mise.jdx.dev) をツール / ランタイムバージョンマネージャとして必須利用します。[公式インストール手順](https://mise.jdx.dev/getting-started.html) で mise をインストールしたあと、**shell init に mise activate を仕込むことが必須です**（任意ではありません）。Make ターゲットは `golangci-lint` / `lefthook` 等を mise の shim 経由で解決しており、activate しない限り shim が `PATH` に載らないためです:

```sh
# zsh
echo 'eval "$(mise activate zsh)"' >> ~/.zshrc

# bash
echo 'eval "$(mise activate bash)"' >> ~/.bashrc

# shell を再起動 (or 新しいターミナルを開く)
exec $SHELL
```

確認:

```sh
mise --version
which mise
```

### 1.2. Go ランタイムとプロジェクトツールのインストール

全ツール（golangci-lint / sqlc / oapi-codegen / mockgen / dlv / lefthook / ...）のバージョンは [`mise.toml`](../../../mise.toml) を SSOT として管理しています。Dockerfile・ローカルインストーラ (`.makefiles/go/installer.mk`)・CI ワークフローはすべて同じ `mise.toml` を参照し、各環境で必要なものだけを `mise install <tool>` で個別取得します。

```sh
make go-update       # Go ランタイムを mise.toml 記載の pin でインストール
make install-tools   # gopls / gotests / impl / dlv / lefthook / golangci-lint をインストール
make activate-tools  # `lefthook install` で git hooks を有効化
```

## Phase 2: ローカル起動確認

ローカルで起動してみて、問題なく動作することを確認してください。

```sh
make serve
make tools
make db-init
```

## Phase 3: ローカライゼーションスクリプトの実行

下記コマンドで、Goモジュール名を一括置換するスクリプトを実行してください。

ORGとREPOは適宜置き換えてください。派生設定は気になる箇所のみ変更してください。

```sh
export ORG=<your-org/git-user-name>
export REPO=<your-repo>

export MODULE=${REPO}
export APP_NAME=${REPO}
export OPENAPI_TITLE=${REPO}
export COPILOT_TITLE=${REPO}
export COPYRIGHT_HOLDER=${ORG}
export COPYRIGHT_YEAR=$(date +%Y)

make setup-replace-module OLD_MODULE=go-boilerplate NEW_MODULE=$MODULE
make setup-replace-repository-reference REPOSITORY=$ORG/$REPO
make setup-replace-app-metadata APP_NAME=$APP_NAME OPENAPI_TITLE="$OPENAPI_TITLE" COPILOT_TITLE="$COPILOT_TITLE"
make setup-replace-license-copyright COPYRIGHT_HOLDER="$COPYRIGHT_HOLDER" COPYRIGHT_YEAR=$COPYRIGHT_YEAR
make gen-api
make gen-sqlc
make tidy-lib
```

## Phase 4: ローカライゼーションの検証

テストと静的解析、コード生成、ヘルスチェックなど、基本的な機能が問題なく動作することを確認してください。

```sh
make test
make lint
make gen
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Phase 5: 手動書き換え

1. [README.md](README.md), [README.ja.md](README.ja.md) の内容をプロジェクトに合わせて書き換えてください。
2. [README.md](README.md) は英語で書かれているので、必要に応じて [README.ja.md](README.ja.md) を [README.md](README.md) に置換しても構いません。
    - ただし、[gen-docs-json.mjs](scripts/gen-docs-json.mjs) やその生成元になる [manifest.yaml](docs/portal/manifest.yaml) などのドキュメント生成スクリプトはREADME.mdを参照しているため、完全に置換する場合はこれらのスクリプトも書き換える必要があります。
    - また、portal表示のReactも EnとJp切り替えを持つので、README.mdを日本語にする場合は、portal表示のReactも書き換える必要があります。
3. [openapi.yaml](openapi/openapi.yaml) の内容をプロジェクトに合わせて書き換えてください。
    - Infoセクション全体をプロジェクトに合わせて書き換えてください。
        - title
        - termsOfService
        - contact
        - version
        - description
        - license

## Phase 6: envファイルの書き換え

[env/](env/) ディレクトリ内のファイルをプロジェクトに合わせて書き換えてください。

設定値の意味については、[env/README.ja.md](env/README.ja.md) を参照してください。

## Phase 7: リポジトリの初期化

ここまでの手順を完了したら、ファーストプッシュ後にリポジトリの初期化を行います。

### GitHubテンプレートから始めた場合

```sh
git add -A
git commit -m "Initial commit: setup boilerplate for $REPO"
git push origin main
make setup-repo
make branch-minor
```

### Git Cloneから始めた場合

```sh
git remote set-url origin ${ORG}/${REPO}
git add -A
git commit -m "Initial commit: setup boilerplate for $REPO"
git push -u origin main
make setup-repo
make branch-minor
```

## Phase 8: デプロイ設定の作成

このboilerplateでは、各社・各個人のクラウド環境やオンプレ環境に合わせて柔軟にデプロイできるよう、特定のクラウドプロバイダやデプロイ方法に依存しない構成を採用しています。

そのため、デプロイ設定には具体的なデプロイ先が反映されていません。プロジェクトのデプロイ先に合わせて、必要な設定を追加してください。

デプロイCI/CD: [.github/workflows/deploy-app.yaml](.github/workflows/deploy-app.yaml) を完成させてください。

`Note: Please modify this section according to your environment` と書かれている箇所が、環境に合わせて変更が必要な箇所になります。

## Phase 9: 認証機の作成

このboilerplateには、認証機能の実装例として、JWTを使用したサンプルコードが含まれています。プロジェクトの要件に合わせて、認証機能を実装してください。

usecaseの[Authenticator](internal/usecase/boundary/auth/authenticator.go)インターフェースを実装する形で、認証機能を作成します。

実装は [internal/infrastructure/auth/README.ja.md](internal/infrastructure/auth/README.ja.md) を参照してください。

実装例(local): [internal/infrastructure/auth/local/auth_local.go](internal/infrastructure/auth/local/auth_local.go)

実装が完了したら、[認証のDIモジュール](internal/di/module/core/auth.go) を編集して、認証機能をアプリケーションに組み込んでください。

## Phase 10: サンプルAPIの削除

このboilerplateには、サンプルAPIが含まれています。プロジェクトの要件に合わせて、サンプルAPIを削除してください。

AI駆動開発を活用する場合は、サンプルAPIを残しておくと、AIがコードの構造や実装例を理解しやすくなります。必要に応じて、サンプルAPIをリファクタリングして、プロジェクトの要件に近づけることもできます。

### 削除手順

自動コマンドを使用します。[scripts/setup/lib/sample-api.mjs](scripts/setup/lib/sample-api.mjs) に宣言されたサンプルAPI（`user` / `product` / `order`）を削除し、共有ファイル（DI 4 モジュール＋ `openapi.yaml`）の `sample-api` マーカーブロックを除去したうえで、再生成・整形・Lint まで実行します。

> 実行前に **DB コンテナが起動している必要があります** — 末尾の `gen-query` は `pg_dump` で**ライブ**スキーマをダンプするため、DB 停止状態では `connection refused` で失敗します。

```bash
# 0. DB コンテナを起動（gen-query がライブスキーマをダンプするため）
docker compose up -d database

# 削除内容のプレビュー（変更は行いません）
DRY_RUN=1 make setup-remove-sample-api

# サンプル削除（ファイル削除＋マーカー除去 → gen-api → gen-query → fix → lint）
make setup-remove-sample-api

# サンプル削除後のマイグレーション集合で DB を再構築しスキーマを再ダンプ
# （削除済みの users テーブルが models.gen.go に残らないようにする）
make db-init-local db-init-test
make gen-query
```

補足:

- 基盤マスタデータ `prefecture`（マイグレーション `000001` など）は**残します**。
- `gen-query` は**ライブ** DB の `pg_dump` から Go モデルを再生成します。上記の DB 再構築を省くと、残存する `users` テーブルが再ダンプされ `models.gen.go` に古い `Users` 型が再生成されます。再構築＋再 `gen-query` が実際に型を消す手順です。
- 共有生成物（`*.gen.go` / `openapi.gen.yaml` など）は直接削除せず、再生成ステップで更新されます。
- サンプルは3ドメイン構成です。`user` はフルスタック、`product` / `order` は現状 DB スタブ（マイグレーション＋商品 seed）のみです。`product` / `order` を本格的な API に拡張したら、`sample-api.mjs` の該当ドメインブロックに新しいパスを追記し、共有ファイル内に混在するサンプル行を `// sample-api:begin` … `// sample-api:end`（または行末の `// sample-api:line`）で囲んでください。同じコマンドで自動的に削除対象に含まれます。

<details>
<summary>手動手順（参考・現在は不要）</summary>

1. [openapi.yaml](openapi/openapi.yaml) のサンプルAPI定義の削除
    - `サンプルAPI用のパス` の下に書かれているPath定義を削除し、そのリンク先のyamlファイルも削除してください。
    - `サンプルAPI用のパラメーター定義` の下に書かれているParameter定義を削除し、そのリンク先のyamlファイルも削除してください。
    - `サンプルAPI用の型定義` の下に書かれているSchema定義を削除し、そのリンク先のyamlファイルも回帰的に削除してください。
2. サンプルAPIのControllerとUsecaseの削除
    1. `make gen-api` でコードを再生成して、サンプルAPIのControllerコードを削除してください。
    2. サンプルAPIが参照している、Usecaseファイルとそのテストファイルを削除してください。
        - mockファイルも削除してください。
    3. [internal/integration](internal/integration/) でエラーを起こしているファイルがあれば、そのファイルも削除してください。
    4. サンプルAPIの生成コードがないことで影響を出しているハンドラファイルおよびテストファイルを削除してください。
    5. この時、Infra層で参照エラー(QueryServiceやCommandServiceのインターフェースエラー)が出る場合は、これらのインターフェースからサンプルAPIで使っているファイルとそのテストコードを削除してください。
3. サンプルAPIのInfraコードの削除
    1. `make db-test-migrate-down` と `make db-local-migrate-down` を実行して、DBをクリーンな状態にする。
    2. `dml` にある実行SQLを削除する。
        - [database/dml/repository](database/dml/repository) の配下のディレクトリを削除してください。
        - [database/dml/query_service](database/dml/query_service) の配下のディレクトリを削除してください。
        - [database/dml/command_service](database/dml/command_service) の配下のディレクトリを削除してください。
    3. `make gen-query` を実行して、SQLCのコードを再生成して、サンプル用のSQLCコードを削除する。
    4. サンプル用のInfraコードがエラーになるので、そのコードとそのテストコードを削除する。
4. サンプルAPIのドメインコードの削除。
    - [internal/domain/](internal/domain/) の配下のサンプルAPIで使っているコードとそのテストコードを削除してください。このディレクトリの配下のディレクトリはサンプルAPIのドメインコードのみなので、配下のディレクトリごと削除しても構いません。

</details>
