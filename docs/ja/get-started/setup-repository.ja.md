# リポジトリ複製後の作業リスト

[English](../../get-started/setup-repository.md) | 日本語

Makeコマンド詳細は [Makeターゲット一覧](../../../.makefiles/README.ja.md) を参照してください。

## Phase 1: mise のインストールとシェル activate

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

## Phase 2: Go ランタイムとプロジェクトツールのインストール

全ツール（golangci-lint / sqlc / oapi-codegen / mockgen / dlv / lefthook / ...）のバージョンは [`mise.toml`](../../../mise.toml) を SSOT として管理しています。Dockerfile・ローカルインストーラ (`.makefiles/go/installer.mk`)・CI ワークフローはすべて同じ `mise.toml` を参照し、各環境で必要なものだけを `mise install <tool>` で個別取得します。

```sh
make go-update       # Go ランタイムを mise.toml 記載の pin でインストール
make install-tools   # gopls / gotests / impl / dlv / lefthook / golangci-lint / zizmor をインストール
make activate-tools  # `lefthook install` で git hooks を有効化
```

## Phase 3: エージェント設定のインストール（推奨構成）

AI 支援レイヤは設定として同梱しています。project スコープの公式プラグイン、本リポジトリ自身のスキル（[`.claude/`](../../../.claude/README.md) / [`.codex/`](../../../.codex/README.md)）、および公式に推奨する外部スキル 1 つ（`graphify`。リポジトリを問い合わせ可能な知識グラフにするツール）です。clone に付いてこない部分は、冪等な bootstrap 2 本で入れます。

```sh
bash .claude/scripts/bootstrap-plugins.sh          # 公式プラグイン（project スコープ）
bash .claude/scripts/bootstrap-external-skills.sh  # 外部スキル（user スコープ: Claude Code + Codex）
```

`graphify` 本体は他のツールと同様に `mise.toml` で pin しているため `mise install` の時点で取得済みで、bootstrap は各アシスタントの設定ディレクトリへ skill を書くだけです。どのコマンドがローカル完結で、どれが LLM API へ出るかは [`.claude/README.md`](../../../.claude/README.md) に記載しています。

**AI 支援レイヤを採らない判断は、導入側のアーキテクトが行います。** 本テンプレートは AI ツール無しでも完全に保守できるよう作られており（レイヤ規約の正本は [docs/rules.md](../../rules.md) であってアシスタント設定ではありません）、上記はビルド・テスト・リリースのいずれにも必須ではありません。採らない fork は、中途半端に設定を残さず意図的に外してください。

- bootstrap 2 本を実行しない（以降のどの Phase もこれらに依存しません）
- 保持しないものを削除する: `.claude/`、`.codex/`、`mise.toml` の `pipx:graphifyy[sql]` pin、`.graphifyignore`、`.gitignore` / `.markdownlint-cli2.yaml` / `scripts/mermaid-lint/index.ts` の `graphify-out/` 記述

後から外すコストは今外すコストと同じなので、まず推奨構成で入れて後から判断する順序でも安全です。

## Phase 4: ローカル起動確認

ローカルで起動してみて、問題なく動作することを確認してください。

```sh
make serve
make tools
make db-init
```

<!-- boilerplate:begin -->
## Phase 5: ローカライゼーションスクリプトの実行

下記コマンドで、Goモジュール名を一括置換するスクリプトを実行してください。

ORG・REPO・CODE_OWNERS は適宜置き換えてください。派生設定は気になる箇所のみ変更してください。

```sh
export ORG=<your-org/git-user-name>
export REPO=<your-repo>

# CODEOWNERS の所有者。ユーザー(@name)かチーム(@org/team)を指定します。
# 組織そのものは所有者になれないため、組織が持つ fork ではチームを指定してください。
export CODE_OWNERS=<@your-org/tech-leads>

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
make setup-replace-codeowners OWNERS="$CODE_OWNERS"
make gen-api
make gen-sqlc
make tidy-lib

# 置換が当たったことを検証し、通ったら初期化ツールを撤去する。
# 最後に実行する。上のスクリプトは一度きりのもので、ここが通るまでは何度でも当て直せる。
make setup-verify
```

`setup-verify` は、`replace-module` が対象と宣言した全ファイルにボイラープレート名が残っていないこと、
および LICENSE・CODEOWNERS・README・OpenAPI が指定した値になっていることを確認する。通った場合にだけ
`scripts/setup/replace-*` と自身を削除する。初期化済みのリポジトリへ当て直すのは誤りで、
`replace-codeowners` は**全ルール**の所有者を単一の値へ書き換えてしまうため。`scripts/setup/lib` は、
一度きりのツール 2 つ（これとサンプル削除）のうち後に走った方が道連れにする。

## Phase 6: ローカライゼーションの検証

テストと静的解析、コード生成、ヘルスチェックなど、基本的な機能が問題なく動作することを確認してください。

```sh
make test
make lint
make gen
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Phase 7: 手動書き換え

1. [README.md](../../../README.md), [README.ja.md](../../../README.ja.md) の内容をプロジェクトに合わせて書き換えてください。
2. [README.md](../../../README.md) は英語で書かれているので、必要に応じて [README.ja.md](../../../README.ja.md) を [README.md](../../../README.md) に置換しても構いません。
    - ただし、[gen-docs-json.ts](../../../scripts/portal/gen-docs-json.ts) やその生成元になる [manifest.yaml](../../../docs/portal/manifest.yaml) などのドキュメント生成スクリプトはREADME.mdを参照しているため、完全に置換する場合はこれらのスクリプトも書き換える必要があります。
    - また、portal表示のReactも EnとJp切り替えを持つので、README.mdを日本語にする場合は、portal表示のReactも書き換える必要があります。
3. [openapi.yaml](../../../openapi/openapi.yaml) の内容をプロジェクトに合わせて書き換えてください。
    - Infoセクション全体をプロジェクトに合わせて書き換えてください。
        - title
        - termsOfService
        - contact
        - version
        - description
        - license

<!-- boilerplate:end -->
## Phase 8: envファイルの書き換え

[env/](../../../env/) ディレクトリ内のファイルをプロジェクトに合わせて書き換えてください。

設定値の意味については、[env/README.ja.md](../../../env/README.ja.md) を参照してください。

## Phase 9: リポジトリの初期化

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

## Phase 10: デプロイ設定の作成

このboilerplateでは、各社・各個人のクラウド環境やオンプレ環境に合わせて柔軟にデプロイできるよう、特定のクラウドプロバイダやデプロイ方法に依存しない構成を採用しています。

そのため、デプロイ設定には具体的なデプロイ先が反映されていません。プロジェクトのデプロイ先に合わせて、必要な設定を追加してください。

デプロイCI/CD: [.github/workflows/deploy-app.yaml](../../../.github/workflows/deploy-app.yaml) を完成させてください。

`Note: Please modify this section according to your environment` と書かれている箇所が、環境に合わせて変更が必要な箇所になります。

## Phase 11: 認証・認可の実装

この boilerplate は認証（authn）と認可（authz）の双方について **開発用スタブのみ** を同梱しており、それらは `local` / `ci` / `test` 環境に **限って** 配線されています。`development` / `staging` / `production` では DI プロバイダが **fail-closed** です。スタブの配線を拒否してエラーを返すため、本物のコンポーネントを実装・配線するまでアプリケーションは **意図的に起動しません**。

これは意図的な強制装置です。署名を検証しない認証器や許可オールの認可器が本番環境に出荷されることを決して起こさないためのものです。**`development` / `staging` / `production` 向けに両方を実装することは、プロジェクト開始時の必須タスクです。**

> [!IMPORTANT]
> `Authorizer` は `InfrastructureModule` の内部で提供されるため、**usecase を構築するすべてのプロセス** — HTTP サーバ **と** バックグラウンドの job / worker プロセス — が設定済みの `Authorizer` を必要とします。後述の認可の手順が完了するまで、`APP_ENV=development` / `staging` / `production` でいずれを起動しても Fx 構築時に `no authorizer configured for environment` で終了します（authn も同様に `no authenticator configured for environment`）。本物のコンポーネントを実装する前にこれが表示されるのは想定内であり、バグではありません。

### 認証（authn）

この boilerplate には認証の実装例として JWT を使用したサンプルコードが含まれています。プロジェクトの要件に合わせて認証を実装してください。

usecase の [Authenticator](../../../internal/usecase/boundary/auth/authenticator.go) インターフェースを実装する形で認証機能を作成します。

- 参照: [internal/infrastructure/auth/README.ja.md](../../../internal/infrastructure/auth/README.ja.md)
- スタブ実装例（local・署名なし）: [internal/infrastructure/auth/local/auth_local.go](../../../internal/infrastructure/auth/local/auth_local.go)
- `stg` / `prd` 実装（JWT / OAuth2 / OIDC / Cognito / Auth0 など）を `internal/infrastructure/auth/{stg,prd}/` 配下に追加します。
- 環境ごとの配線は [認証の DI モジュール](../../../internal/di/module/core/auth.go)（`provideAuthenticator`）を編集し、`default` の fail-closed 分岐を `case config.EnvDevelopment / EnvStaging / EnvProduction` に置き換えて本物の `Authenticator` を返します。

### 認可（authz）

この boilerplate は開発用スタブとして **許可オール（allow-all）** の認可器を同梱しています。自プロジェクト向けに本物の Policy Decision Point（PDP）を実装してください。

usecase の [Authorizer](../../../internal/usecase/boundary/authz/authorizer.go) インターフェースを実装する形で認可機能を作成します。

- 参照: [internal/infrastructure/authz/README.ja.md](../../../internal/infrastructure/authz/README.ja.md)
- スタブ実装例（許可オール）: [internal/infrastructure/authz/allowall/authz_allowall.go](../../../internal/infrastructure/authz/allowall/authz_allowall.go)
- `stg` / `prd` 実装（claims からの RBAC / 所有者チェック / OPA・Cedar などの外部ポリシーエンジン）を `internal/infrastructure/authz/{stg,prd}/` 配下に追加します。
- 環境ごとの配線は [認可の DI モジュール](../../../internal/di/module/authz.go)（`provideAuthorizer`）を編集し、`default` の fail-closed 分岐を `case config.EnvDevelopment / EnvStaging / EnvProduction` に置き換えて本物の `Authorizer` を返します。

`Authorize(ctx, *auth.Authn, Action, *Resource)` のシグネチャは既に完全な `Authn`（subject / scopes / claims）と対象 `Resource`（任意の `OwnerID` 付き）を運ぶため、RBAC と所有者（オブジェクトレベル）モデルの双方を呼び出し箇所を変えずに表現できます。

## Phase 12: ボイラープレートの顔を消す

このリポジトリは数箇所で自分をボイラープレートと呼んでいる（README の 2 節と、本ガイドの
ローカライゼーション手順）。いずれもテンプレートの足場であって、あなたのプロジェクトの
ドキュメントではない。

```sh
DRY_RUN=1 make setup-remove-boilerplate-identity
make setup-remove-boilerplate-identity
```

マークされた記述を落とし、自身の make ターゲットの登録も外したうえで、ツール自身を撤去する。
**触らないもの**: リポジトリ名・モジュール名（Phase 5 で置換済み）と、運用中も読み返す部分
——後述の clamp 設定レビューと除外 ADR。これらは複数のパッケージ README から参照されている。

## Phase 13: テンプレートの意図的な除外（ADR）のレビュー

認証・認可（Phase 11）やデプロイ（Phase 10）以外にも、このテンプレートはいくつかの**意図的な非選択**をしています。例：アプリ内レート制限器を持たない / 汎用 Cache 抽象を持たない / scheduled job の並走制御はスケジューラに委譲 / push・streaming ブローカーは worker の対象外。

これらは [docs/adr/](../../../docs/adr/) 配下の **exclusion ADR** として記録され、`setup-review` タグが付いています。次で一覧できます：

```sh
grep -rl "setup-review" docs/adr/
```

プロジェクトごとに各 ADR をレビューし、次を判断してください：

- **そのまま採用** — その除外が自プロジェクトに合う場合は ADR をそのままにする。
- **変更** — 逆の方針が必要な場合。セットアップはテンプレートから自プロジェクトの**ベースラインを確立**する場なので、**ADR を直接編集**（Decision / Consequences を書き換え、`deciders` / `date` を更新）して自プロジェクトの選択を記録し、実装する。

不変（元 ADR は編集せず supersede する新 ADR を追加）モデルは、**運用開始後**に決定を見直すときに適用します。セットアップ時の一度きりの再ベースライン化には適用しません。

## Phase 14: 依存ライセンス方針の決定

依存ライセンススキャン（`make trivy-license` と [.github/workflows/trivy-fs.yaml](../../../.github/workflows/trivy-fs.yaml) の `trivy-license` ジョブ）は**恒久的に報告専用**です。全依存のライセンスをジョブサマリと PR コメントへ列挙するだけで、ビルドを落とすことはありません。

これは未完成のゲートではなく、**意図的な非選択**です。どのライセンスを許容するかは、このテンプレートを採用する組織が持つ法務判断です。配布されるバイナリでは失格となる copyleft が、バイナリが自社インフラの外へ出ないサービスでは全く問題にならないこともあり、答えは会社・製品・配布形態ごとに変わります。ここで閾値を決めることは、ある一社の法務スタンスを全 fork へ焼き込むことになるため、テンプレートは棚卸しだけを提供し、判断は採用側に委ねます。

自組織に禁止ライセンス方針がある（あるいは必要な）場合は、次のように自分でゲート化してください：

1. 許容する集合を Trivy 自身の分類（`notice` / `unencumbered` / `permissive` / `reciprocal` / `restricted` / `forbidden` / `unknown`）で決め、**出荷物とビルド専用ツールに同じ基準を当てるか**を判断する。同じでなくてよい：本リポジトリで `notice` / `unencumbered` 以外に分類される依存は、出荷されないビルド専用の `docker/tools/` 由来です。
2. Trivy の分類は出発点であり権威ではないものとして扱う。`BlueOak-1.0.0` は OSI 承認の permissive ライセンスでありながら `unknown` に落ちるため、この種のケースは分類任せにせず明示的に決める。
3. [.makefiles/security/trivy.mk](../../../.makefiles/security/trivy.mk) の `trivy-license-ci` へ閾値を追加し、`trivy-license` ジョブへ失敗させるステップを足す。パッケージ単位の例外は [.trivyignore.yaml](../../../.trivyignore.yaml) へ記録する。
4. [.github/workflows/README.md](../../../.github/workflows/README.md) のトリガーマトリクスと [ADR-0084](../../../docs/adr/0084-multi-layer-security-scanning.md) のライセンス行を更新する（いずれも現状「方針なし」と記載しています）。

## Phase 15: サンプルAPIの削除

このboilerplateには、サンプルAPIが含まれています。プロジェクトの要件に合わせて、サンプルAPIを削除してください。

AI駆動開発を活用する場合は、サンプルAPIを残しておくと、AIがコードの構造や実装例を理解しやすくなります。必要に応じて、サンプルAPIをリファクタリングして、プロジェクトの要件に近づけることもできます。

### 削除手順

自動コマンドを使用します。[scripts/setup/remove-sample-api/sample-manifest.ts](../../../scripts/setup/remove-sample-api/sample-manifest.ts) に宣言されたサンプルAPI（`user` / `product` / `order`）を削除し、共有ファイル（DI 4 モジュール＋ `openapi.yaml`）の `sample-api` マーカーブロックを除去したうえで、再生成・整形・Lint まで実行します。

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
- サンプルは3ドメイン構成です。`user` はフルスタック、`product` / `order` は現状 DB スタブ（マイグレーション＋商品 seed）のみです。`product` / `order` を本格的な API に拡張したら、`sample-manifest.ts` の該当ドメインブロックに新しいパスを追記し、共有ファイル内に混在するサンプル行を `// sample-api:begin` … `// sample-api:end`（または行末の `// sample-api:line`）で囲んでください。同じコマンドで自動的に削除対象に含まれます。

### 規則から例が消えるので、自分の例を置き直す

`docs/rules.md`・`docs/adr/**`・各層 `README` のいくつかの規則は、一般的な形で述べたうえでサンプル由来の
具体例で説明しています。**規則は撤去後も残りますが、例は残りません。** 残るのは正しい記述と、それが自分の
システムでどう見えるかを示さなくなった状態です。

該当箇所には、消える行の直上に HTML コメントで**なぜそこに例が要るのか・その例が何を示さなければならないか・
どう書き直すか**が置いてあります。探して順に埋めてください。

```bash
grep -rn "撤去後にこの箇所へ自分の例を置くための指針" docs/ internal/ pkg/
```

これは見栄えの話ではありません。例の無い抽象的な規則は、人が規則を適用しなくなる直前の姿です。読者それぞれが
何が対象なのかを一人で判断することになり、判断は割れます。コメントは、その判断を元の意図が見えているうちに
一度だけ下せるように置いてあります。

業務語彙には専用の家があります。[`docs/spec/glossary.md`](../../spec/glossary.md) の用語表も同じ撤去で空に
なりますが、埋め直すための規則はページに残ります。

<details>
<summary>手動手順（参考・現在は不要）</summary>

1. [openapi.yaml](../../../openapi/openapi.yaml) のサンプルAPI定義の削除
    - `サンプルAPI用のパス` の下に書かれているPath定義を削除し、そのリンク先のyamlファイルも削除してください。
    - `サンプルAPI用のパラメーター定義` の下に書かれているParameter定義を削除し、そのリンク先のyamlファイルも削除してください。
    - `サンプルAPI用の型定義` の下に書かれているSchema定義を削除し、そのリンク先のyamlファイルも回帰的に削除してください。
2. サンプルAPIのControllerとUsecaseの削除
    1. `make gen-api` でコードを再生成して、サンプルAPIのControllerコードを削除してください。
    2. サンプルAPIが参照している、Usecaseファイルとそのテストファイルを削除してください。
        - mockファイルも削除してください。
    3. [internal/integration](../../../internal/integration/) でエラーを起こしているファイルがあれば、そのファイルも削除してください。
    4. サンプルAPIの生成コードがないことで影響を出しているハンドラファイルおよびテストファイルを削除してください。
    5. この時、Infra層で参照エラー(QueryServiceやCommandServiceのインターフェースエラー)が出る場合は、これらのインターフェースからサンプルAPIで使っているファイルとそのテストコードを削除してください。
3. サンプルAPIのInfraコードの削除
    1. `make db-test-migrate-down` と `make db-local-migrate-down` を実行して、DBをクリーンな状態にする。
    2. `dml` にある実行SQLを削除する。
        - [database/dml/repository](../../../database/dml/repository) の配下のディレクトリを削除してください。
        - [database/dml/query_service](../../../database/dml/query_service) の配下のディレクトリを削除してください。
        - [database/dml/command_service](../../../database/dml/command_service) の配下のディレクトリを削除してください。
    3. `make gen-query` を実行して、SQLCのコードを再生成して、サンプル用のSQLCコードを削除する。
    4. サンプル用のInfraコードがエラーになるので、そのコードとそのテストコードを削除する。
4. サンプルAPIのドメインコードの削除。
    - [internal/domain/](../../../internal/domain/) の配下のサンプルAPIで使っているコードとそのテストコードを削除してください。このディレクトリの配下のディレクトリはサンプルAPIのドメインコードのみなので、配下のディレクトリごと削除しても構いません。

</details>

## Phase 16: ボイラープレート限定の規約の削除

[boilerplate-only-conventions.ja.md](boilerplate-only-conventions.ja.md) は、本リポジトリが上流のテンプレートである間だけ成り立つ規約——ADR のその場改訂 regime、統合パス、`setup-review` の仕掛け——を集約したものです。いずれもあなたのプロジェクトには適用されず、各規則の一般形はそれを所有する文書のほうに既に書かれています。

サンプルAPI（Phase 15）を残す場合でも、この Phase は実施してください。2 つの削除は独立しており、発火する契機が違います。

1. ファイルと日本語ミラーを削除します。

    ```sh
    rm docs/get-started/boilerplate-only-conventions.md \
       docs/ja/get-started/boilerplate-only-conventions.ja.md
    ```

2. `boilerplate-only:line` マーカーを持つ行をすべて除去します。いずれも削除対象ファイルへの自己完結したポインタ 1 行なので、除去しても周囲の文は壊れません。

    ```sh
    grep -rn "boilerplate-only:line" docs/
    ```

3. 自分の ADR regime を決めます。継承されるのは [docs/adr/README.md](../../adr/README.md)（日本語は [README.ja.md](../adr/README.ja.md)）に書かれたとおりのもの——ADR は不変の記録であり、変わった決定は新しい `accepted` な ADR で置き換え、古いものは `superseded` とする——です。その場改訂を望むなら、それはあなた自身の決定として、あなた自身の ADR に記録してください。

> マーカー名前空間 `boilerplate-only` は**暫定**で、この削除は当面手作業です（これを扱う strip スクリプトは別途準備中）。`boilerplate-only` と `sample-api` のマーカーを 1 回のパスで剥がさないでください。発火する契機が違い、fork が片方だけを行うことは十分あり得ます。

<!-- dast:begin -->
## Phase 17: DAST のセットアップを残すかを決める

DAST のセットアップだけ済ませてあります。[`.github/workflows/zap-api-scan.yaml`](../../../.github/workflows/zap-api-scan.yaml) は GitHub-hosted runner の中でこのアプリケーションを起動し、OpenAPI 定義から得たエンドポイント一覧をもとに、認証済みの [OWASP ZAP](https://www.zaproxy.org/) API スキャンを当てます。週次と手動で走り、結果は code scanning へ上がり、検出でビルドを落とすことはありません。動的スキャンが要るならそのまま使ってください。追加で配線するものはありません。

**ただし設定値はサンプルです。** [`.github/zap/rules.tsv`](../../../.github/zap/rules.tsv) のしきい値、スキャンが名乗るアイデンティティ、そのスキャンが到達する面は、いずれもこの boilerplate のサンプル API と `ci` 環境プロファイルに合わせた仮の値であり、あなたの API について何かを主張するものではありません。エンドポイントが違えば、受容してよい検出も違います。最初の週次実行の前にワークフロー冒頭のコメントを読み、実際の API に合わせて両方のファイルを調整してください。引き継いだままの `IGNORE` は、二度と誰の目にも触れない検出になります。

サンプル API との結合は 1 箇所だけ残してあり、そこは黙らず落ちる作りにしてあります。ワークフローは ZAP を起動する前に、保護されたサンプルエンドポイントを叩いてスキャンが認証済みであることを確かめます。したがってサンプル API を削除（Phase 15）したまま DAST を残すと、`PROBE_PATH` を自分のオペレーションに向け直すまで週次実行が赤くなります。これは意図した挙動です。対象が消えた認証確認をそのままにすると、スキャンが 401 しか見ていないのに緑を報告し続けることになります。

要らなければ、丸ごと撤去してください。

```sh
# 撤去対象のプレビュー（何も変更しません）
docker compose run --rm node_tool_runner pnpm --dir scripts run tsx \
  scripts/setup/remove-dast-setting --dry-run

# 撤去
docker compose run --rm node_tool_runner pnpm --dir scripts run tsx \
  scripts/setup/remove-dast-setting
```

ワークフロー本体・ZAP のルールファイル・[.github/workflows/README.md](../../../.github/workflows/README.md) とその日本語ミラーの該当行・この節・pin lockfile に残るスキャナ用 action のエントリ、そして最後にツール自身を削除します。有効/無効を切り替えるスイッチはありませんし、今後も設けません。残すとは残すことであり、設定されたまま無効なスキャナは、誰も読まず誰も保守しないものになります。

撤去後に中身を参照したくなったら git の履歴から辿れます。撤去はコミット 1 つで、`git log -- .github/workflows/zap-api-scan.yaml` で見つかります。
<!-- dast:end -->
