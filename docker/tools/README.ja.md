# Tools コンテナ

[English](README.md) | 日本語

プロジェクトの**コード生成・バンドル用ツールコンテナ**を定義する Dockerfile です。マルチステージビルドにより Go / Node.js / Python のツール環境を提供します。

## ビルドターゲット

|ターゲット|ベースイメージ|含まれるツール|
|---|---|---|
|`go_tools`|`golang:1.26.3-alpine`|oapi-codegen, mockgen, sqlc, migrate|
|`node_tools`|`node:24.14-alpine`|redocly-cli, js-yaml|
|`python_tools`|`python:3.14.2-slim`|sqlfluff|

## go_tools

Go 用のコード生成ツール：

|ツール|用途|
|---|---|
|`oapi-codegen`|OpenAPI 仕様から Go サーバー / 型を生成|
|`mockgen`|Go interface からモックを生成|
|`sqlc`|SQL から型安全な Go コードを生成|
|`migrate`|データベースマイグレーション CLI|

## node_tools

OpenAPI ドキュメント処理ツール：

|ツール|用途|
|---|---|
|`redocly-cli`|OpenAPI YAML のバンドル（`$ref` 解決）と HTML ドキュメント生成|
|`js-yaml`|ポータルドキュメント生成スクリプト用の YAML 処理|

## python_tools

SQL リンティングツール：

|ツール|用途|
|---|---|
|`sqlfluff`|migration / DML / seed ファイル用の SQL リンター|

## docker-compose サービス

```yaml
go_tool_runner:     # target: go_tools,    profile: generate
node_tool_runner:   # target: node_tools,  profile: generate
python_tool_runner: # target: python_tools, profile: generate
```

すべてのツールランナーはプロジェクトルートを `/app` にマウントし、root で実行します。

## 実行方法

```bash
make gen        # すべてのコード生成を実行
make gen-api    # OpenAPI バンドル + Go コード生成
make gen-query  # sqlc コード生成
```

## ツールを追加する場合

1. 適切な Dockerfile ステージにツールをインストール
2. `docker-compose.yaml` に `profiles: [generate]` で新しいサービスを追加
3. 必要に応じて Makefile ターゲットを追加

## 注意点

- すべてのターゲットで作業ディレクトリは `/app`
- Go ツールはビルダーステージでインストールし、ランタイムステージにコピーしてイメージサイズを最小化（`go_tools`）
- 開発初期は `@latest` を使用 — CI 環境ではバージョンを固定すること
