# go-boilerplate

Golang × Echo × OpenAPI × PostgreSQL × Onion Architecture によるベースプロジェクトです。

`uber/fx` による DI や `sqlc`, `golang-migrate`, `oapi-codegen` などを採用しています。

## 本テンプレートの前提

このBoilerplateは、PoC〜新規構築期における初期アーキテクチャ整備を支援する目的で提供されています。  

学習用や社内向けのPoCなどであれば、**そのまま利用しても問題ありません**。

本番で運用する場合は**以下の前提に合致するチーム/リーダー向けのテンプレート**です。

- Go + Echo + Fx + OpenAPI + sqlc などの構成を理解している
- `.env` 管理方針とセキュリティポリシーの線引きが判断できる
- 初期構築の意思決定ができる（=TL相当）

上記を満たさない場合、テンプレートの誤用・理解不足による事故が起きる可能性があります。

## Gitコミットメッセージ用プレフィックス一覧

| プレフィックス | 説明 |
| ------------ | --- |
| `Feat` | 新しい機能追加 |
| `Fix` | バグ修正 |
| `Docs` | ドキュメントのみの変更 |
| `Style` | フォーマット、スペース、セミコロンの修正など（動作に影響なし） |
| `Refactor` | リファクタリング（機能追加やバグ修正を含まないコード改善） |
| `Perf` | パフォーマンス改善 |
| `Test` | テストの追加・修正 |
| `Build` | ビルドシステムや依存関係に関する変更 |
| `Ci` | CI 設定やスクリプトの変更 |
| `Chore` | その他の雑多な変更（ビルド以外の補助タスク） |
| `Revert` | コミットの取り消し |

## 利用ツール(サポートバージョン)

- Go(1.24.4)
- Docker Desktop
- Github CLI
- Postman

<details>
<summary>手動インストール先のURL</summary>

下記のサイトからダウンロードして進めてください。

- [Golang](https://go.dev/dl/)
- [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/)
- [Github CLI](https://cli.github.com/)
- [Postman](https://www.postman.com/downloads/)

</details>

<details>
<summary>brewでのインストール方法</summary>

コピペで実行できます。

```bash
# anyenvのインストール
brew install anyenv
anyenv init
echo 'eval "$(anyenv init -)"' >> ~/.zprofile

# anyenvのupdateプラグインのインストール
mkdir -p $(anyenv root)/plugins
git clone https://github.com/znz/anyenv-update.git $(anyenv root)/plugins/anyenv-update
anyenv update

# goenvのインストール
anyenv install goenv
goenv install "$(cat .go-version)"

# dockerのインストール
brew install --cask docker

# Github CLIのインストール
brew install gh

# Postmanのインストール
brew install --cask postman
```

</details>

## 構成スタック

- **言語**: Go
- **Webフレームワーク**: Echo
- **DI**: uber/fx
- **API定義**: OpenAPI
  - **コード生成**: oapi-codegen
- **DB**: PostgreSQL
- **ORM/Query**: sqlc
- **マイグレーション**: golang-migrate
  - **マイグレーション統合**: tern
- **開発補助**:
  - godotenv
  - zap
  - testify
  - cobra（CLI）
  - air（ホットリロード）
  - Docker / docker-compose

## 開発ドキュメント

```bash
make tools
```

<http://localhost:8082/index.html> にアクセスすると、開発ドキュメントが表示されます。

現在の開発ドキュメントは以下の内容を含みます。

- OpenAPI ドキュメント
- コードカバレッジ
- ER図

## ディレクトリ構成

<details>
<summary>展開する</summary>

```text
 ./
├──  cmd/ # main.go が配置されるディレクトリ
├──  database/ # データベース関連のファイルを配置
│   │
│   ├──  dml/ # データ操作言語 (DML) スクリプトを配置
│   │   ├──  query_service/ # クエリサービスのsqlc用のsqlファイルを配置
│   │   └──  repository/ # リポジトリのsqlc用のsqlファイルを配置
│   │
│   ├──  migrations/ # DDLとマスタデータを持つマイグレーションsqlファイルを配置
│   ├──  seed/ # 開発環境での初期データを投入するためのsqlファイルを配置
│   └──  sqlc/ # SQLCでの設定ファイルを配置
│
├──  docker/ # Dockerfileとそれぞれの役割の設定ファイルなどを配置
│   └──  <Dockerの役割>/ # 各種Dockerfileを配置
│
├──  docs/
│   ├──  coverage/ # 生成されたテストカバレッジレポート
│   ├──  er-diagram/ # 生成されたER図
│   ├──  openapi/ # 生成されたOpenAPI仕様の定義
│   └──  index.html # 全体のドキュメントのためのルーティングファイル
│
├──  internal/
│   │
│   ├──  apperror/ # アプリケーションの基底エラーを定義するパッケージ
│   │
│   ├──  cli/
│   │   ├──  <各種CLI>/ # cliコマンドごとのディレクトリを作成する
│   │   └──  cli.go # このディレクトリのcliを統合するためのファイル
│   │
│   ├──  config/ # アプリ全体で使うコンフィグ設定の生成
│   │
│   ├──  controller/ # コントローラー層
│   │   │
│   │   ├──  ctxhelper/ # コンテキストに特定の項目を設定・取得するためのヘルパーパッケージ
│   │   ├──  error/
│   │   │   └──  response/ # エラーレスポンスを生成するためのパッケージ
│   │   ├──  handler/ # ハンドラーの実装(URIと同じ構成にする)
│   │   ├──  httpstack/ # サーバの拡張機能を提供するパッケージ
│   │   │   └──  extension.go # 拡張機能を適用するためのファイル
│   │   └──  server/ # サーバーの起動や本体の処理
│   │
│   ├──  di/ # 依存性注入(DI)
│   │   ├──  config.go # コンフィグ設定のDI
│   │   ├──  db.go # DB接続のDI
│   │   ├──  handler.go # コントローラ層のDI
│   │   ├──  httpstack.go # サーバー拡張機能のDI
│   │   ├──  logging.go # ロギングのDI
│   │   ├──  repository.go # インフラ層のDI
│   │   ├──  serve.go # サーバーのDI
│   │   └──  usecase.go # ユースケース層のDI
│   │
│   ├──  domain/ # ドメイン層
│   │
│   ├──  infrastructure/ # インフラストラクチャ層
│   │   └──  rdb/ # RDBの実装
│   │       ├──  conv/ # sql専用の型との変換処理
│   │       ├──  driver/ # RDBの接続ドライバの実装
│   │       ├──  repository/ # リポジトリの実装
│   │       ├──  queryservice/ # クエリサービスの実装
│   │       └──  sqlc/ # SQLCの生成物の自動配置場所
│   │
│   ├──  logging/ # ロギング関連
│   │
│   └──  usecase/ # ユースケース関連
│       ├──  paging/ # ページング関連
│       └──  tx/ # トランザクション(インターフェイス)関連
│
├──  openapi/ # OpenAPI仕様の定義
│   │
│   ├──  components/
│   │   ├──  parameters/
│   │   │   └──  pagination/ # 共通で使うページネーションのパラメータ
│   │   │       ├──  PageParam.yaml
│   │   │       └──  PerPageParam.yaml
│   │   ├──  requests/ # APIリクエストの定義(定義を配置するのはschema)
│   │   ├──  responses/ # APIレスポンスの定義(定義を配置するのはschema)
│   │   └──  schemas/ # 共通のスキーマ定義
│   │
│   ├──  paths/
│   │   ├──  internal/ # 特定のoapi-codegenの生成物を利用するためのパス
│   │   │   └──  types/
│   │   │       └──  error_response.yaml
│   │   ├──  v1/
│   │   │   ├──  users/
│   │   │   │   └──  user_id.yaml  # http:localhost/v1/users/{user_id} に対応するファイル
│   │   │   └──  users.yaml # http:localhost/v1/users に対応するファイル
│   │   └──  health.yaml # http:localhost/v1/health に対応するファイル
│   │
│   ├──  openapi.gen.yaml # 自動生成されたOpenAPI仕様
│   └──  openapi.yaml
│
├──  pkg/ # 全体で使う汎用的なパッケージ
├──  scripts/ # 自動生成などで使うスクリプトの配置場所
├──  tmp/ # airを使うときのキャッシュディレクトリ
├──  docker-compose.yaml
├──  go.mod
├──  go.sum
├──  makefile
└── 󰂺 README.md
```

</details>

## 開発開始手順

```bash

make install
make serve
make tools
make db-init

```

## リリース作業手順

### タグ打ち

<details>
<summary>タグ打ちコマンド</summary>

majorバージョンのタグ打ちとlatestのリリースノートを同期

```bash
make release-major-tag
```

minorバージョンのタグ打ちとlatestのリリースノートを同期

```bash
make release-minor-tag
```

patchバージョンのタグ打ちとlatestのリリースノートを同期

```bash
make release-patch-tag
```

</details>

### 次の開発(リリース)ブランチの作成

<details>
<summary>デフォルトブランチ変更コマンド</summary>

majorバージョンで更新したリリースブランチの作成

```bash
make release-major-branch
```

minorバージョンで更新したリリースブランチの作成

```bash
make release-minor-branch
```

patchバージョンで更新したリリースブランチの作成

```bash
make release-patch-branch
```

hotfixブランチの作成

```bash
make hotfix-patch-branch
```

</details>

## リポジトリセットアップ手順

```bash
make setup-repo
```
