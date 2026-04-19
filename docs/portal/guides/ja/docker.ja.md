# docker

[English](README.md) | 日本語

`docker/` は、開発・ビルド・運用に必要な **Docker 関連の設定ファイル**を格納するディレクトリです。

## ディレクトリ構成

```text
docker/
├── server/             # アプリケーションサーバ用 Dockerfile
├── tools/              # コード生成・ツールランナー用 Dockerfile
├── document/           # ドキュメントビューア用 Dockerfile + nginx設定
└── database/
    ├── sql/            # DB初期化SQL
    ├── schemaspy/      # ER図生成設定
    └── sqlfluff/       # SQLリンター設定
```

## サービス概要

docker-compose.yaml で定義されるサービスと、対応する Dockerfile / 設定の対応表です。

### 開発環境（profile: `development`）

|サービス|Dockerfile / Image|ポート|説明|
|---|---|---|---|
|`api_server`|`docker/server/Dockerfile` (target: `tooling`)|8080, 2345, 6060|開発用APIサーバ（air によるホットリロード）|
|`database`|`postgres:18.3-bookworm`|5432|PostgreSQL データベース|

### 補助サービス（profile: `tools`）

|サービス|Dockerfile / Image|ポート|説明|
|---|---|---|---|
|`docs_viewer`|`docker/document/Dockerfile`|8082|ドキュメントポータル（nginx）|
|`sql_editor`|`sosedoff/pgweb`|8081|Web SQL エディタ|

### ツールランナー（profile: `generate`）

|サービス|Dockerfile / Image|説明|
|---|---|---|
|`go_tool_runner`|`docker/tools/Dockerfile` (target: `go_tools`)|oapi-codegen, mockgen, sqlc, migrate|
|`node_tool_runner`|`docker/tools/Dockerfile` (target: `node_tools`)|redocly-cli|
|`python_tool_runner`|`docker/tools/Dockerfile` (target: `python_tools`)|sqlfluff|
|`er_diagram_generator`|`schemaspy/schemaspy`|ER図生成（SchemaSpy）|

## server

アプリケーションサーバの Dockerfile です。マルチステージビルドで以下のターゲットを提供します。

|ターゲット|用途|ベースイメージ|
|---|---|---|
|`builder`|Goバイナリのビルド|`golang:1.26.2-alpine`|
|`runtime`|本番実行用コンテナ|`alpine:3.23`|
|`migration`|マイグレーション実行用コンテナ|`runtime` を継承|
|`tooling`|ローカル開発環境|`golang:1.26.2-alpine`|

### runtime

- 非rootユーザー（`app`）で実行
- `ldflags` でバージョン / リビジョン / ビルド日時を埋め込み
- `vendor` モードでビルド（`GOPROXY=off`）

### tooling

ローカル開発用に以下のツールを含みます。

|ツール|用途|
|---|---|
|`air`|ホットリロード|
|`dlv`|デバッガ|
|`golines`|行長制限付き整形|
|`gofumpt`|gofmt 強化版|
|`golangci-lint`|Go リンター|

## tools

コード生成・バンドル用のツールコンテナです。3つのステージに分かれています。

|ステージ|ベース|含まれるツール|
|---|---|---|
|`go_tools`|`golang:1.26.2-alpine`|oapi-codegen, mockgen, sqlc, migrate|
|`node_tools`|`node:24.14-alpine`|redocly-cli, js-yaml|
|`python_tools`|`python:3.14.2-slim`|sqlfluff|

ツールを追加した場合は、バージョン情報を記録するスクリプトの修正が必要です。詳細は `docs/ja/maintenance/versions_generator.ja.md` を参照してください。

## document

ドキュメントポータル用のコンテナです。

- ベースイメージ: `nginx:1.29-otel`
- `docs/` ディレクトリ全体をボリュームマウント
- ポータルアプリは `/portal/` で提供
- `http://localhost:8082/` にアクセスすると `/portal/` にリダイレクト

## database

### sql

DB初期化用のSQLファイルを格納します。

- PostgreSQL コンテナ起動時に `docker-entrypoint-initdb.d` 経由で実行
- ファイル名の連番プレフィックス順に実行（例: `001-...`, `002-...`）
- DDL（テーブル定義）はここに置かず、`database/migrations/` で管理

### schemaspy

ER図生成ツール（SchemaSpy）の接続設定です。

|ファイル|用途|
|---|---|
|`schemaspy.properties`|ローカル環境用（host: `database`）|
|`schemaspy-ci.properties`|CI環境用（host: `localhost`）|

### sqlfluff

SQLリンター（sqlfluff）の設定ファイルです。対象ごとに異なるルールを適用します。

|ファイル|対象|特徴|
|---|---|---|
|`.dml.sqlfluff`|`database/dml/`（sqlc クエリ）|`@param` プレースホルダ対応、一部ルール除外|
|`.migrations.sqlfluff`|`database/migrations/`|行長制限150|
|`.seed.sqlfluff`|シードデータ|行長制限500|

共通ルール

- dialect: `postgres`
- キーワード: 大文字
- 識別子: 小文字
- 関数 / リテラル / 型: 大文字
