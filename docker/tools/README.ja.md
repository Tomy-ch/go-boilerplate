# Tools コンテナ

[English](README.md) | 日本語

プロジェクトの**コード生成・バンドル用ツールコンテナ**を定義する Dockerfile です。マルチステージビルドにより Go / Node.js / Python のツール環境を提供します。

## 役割

`docker/tools/Dockerfile` はビルドで必要となるすべてのコード生成・lint・セキュリティ・ドキュメント生成ツール（oapi-codegen / mockgen / sqlc / migrate / trivy / actionlint / hadolint / gitleaks / godoc / godoc-static / redocly-cli / markdownlint-cli2 / js-yaml / sqlfluff）を、言語ごとに隔離したランナーイメージにパッケージングします。開発者と CI はこれらのコンテナを `make` ターゲット（`make gen-api` / `make gen-query` / `make sql-lint` 等）経由で起動するため、誰も Go / Node / Python ツールチェインをローカルにインストールする必要がありません。マシン間でツールバージョンが再現可能になり、生成物が既知のツールチェインに固定されます。

## ビルドターゲット

|ターゲット|ベースイメージ|含まれるツール|
|---|---|---|
|`go_tools`|`golang:1.26.4-alpine`|oapi-codegen, mockgen, sqlc, migrate, trivy, actionlint, hadolint, gitleaks, godoc, godoc-static|
|`node_tools`|`node:24.14-alpine`|redocly-cli, markdownlint-cli2, js-yaml, esbuild（+ ポータルバンドル用ライブラリ）|
|`python_tools`|`python:3.14.2-slim`|sqlfluff|

## go_tools

Go 用のコード生成・lint・セキュリティ・ドキュメント生成ツール：

|ツール|用途|
|---|---|
|`oapi-codegen`|OpenAPI 仕様から Go サーバー / 型を生成|
|`mockgen`|Go interface からモックを生成|
|`sqlc`|SQL から型安全な Go コードを生成|
|`migrate`|データベースマイグレーション CLI|
|`trivy`|脆弱性・設定ミスのスキャナー|
|`actionlint`|GitHub Actions ワークフローの Lint|
|`hadolint`|Dockerfile の Lint|
|`gitleaks`|コミットされた認証情報を検出するシークレットスキャナー|
|`godoc`|Go パッケージドキュメントの配信 / 生成|
|`godoc-static`|godoc 出力から静的 HTML を生成|

## node_tools

OpenAPI ドキュメント処理とポータルフロントエンドのバンドル用ツール：

|ツール|用途|
|---|---|
|`redocly-cli`|OpenAPI YAML のバンドル（`$ref` 解決）と HTML ドキュメント生成|
|`markdownlint-cli2`|ドキュメント用の Markdown リンター（`make md-lint`）|
|`js-yaml`|ポータルドキュメント生成スクリプト用の YAML 処理|
|`esbuild`|ポータルフロントエンド（`docs/portal/src/main.jsx`）を `docs/portal/dist/` へバンドル（`make gen-portal-build`）|
|`react` / `react-dom` / `marked` / `fuse.js` / `mermaid` / `highlight.js`|esbuild がバンドルするポータルフロントエンドの実行時ライブラリ（従来の CDN + ブラウザ内 Babel 構成を置き換え）。`mermaid` は `scripts/mermaid-lint.mjs` でも再利用し ` ```mermaid ` フェンスを構文検証する（`make md-lint`）。|
|`linkedom`|`mermaid.parse` を Node で動かすためのヘッドレス DOM。Markdown 内 mermaid の構文 Lint（`scripts/mermaid-lint.mjs`）で使用|

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
- ツールのバージョンは `mise.toml`（バージョンの SSOT）で固定 — 更新はそこで行い、ローカルと CI のイメージを一致させること
