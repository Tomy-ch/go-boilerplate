# scripts

[English](README.md) | 日本語

`scripts/` には、コード生成・ドキュメント・バージョニング・プロジェクト初期設定のための**ユーティリティスクリプト**が格納されています。

## ディレクトリ構成

```text
scripts/
├── gen-docs-json.mjs           # ポータルナビゲーション用 docs.json の生成
├── gen-portal-docs.mjs         # manifest.yaml に基づくドキュメントのポータルへのコピー
├── build-portal.mjs            # ポータルフロントエンド（src/main.jsx）を esbuild でバンドル
├── semver.mjs                  # セマンティックバージョニングヘルパー（patch/minor/major）
├── stamp-openapi-version.mjs   # release/vX.Y.Z のブランチ名から openapi.yaml の info.version を同期
├── sync-versions/              # mise.toml の go / node / python を go.mod と Dockerfile FROM へ反映（Go 実装）
├── make_help.mjs                # Make ターゲットのヘルプ出力生成
├── genctxkey/                  # コンテキストキーコードジェネレータ（Go）
└── setup/                     # プロジェクト初期設定スクリプト
    ├── replace-module.mjs
    ├── replace-app-metadata.mjs
    ├── replace-license-copyright.mjs
    ├── replace-repository-reference.mjs
    ├── remove-sample-api.mjs  # サンプルAPI(user/product/order)を削除
    └── lib/                   # setup スクリプト共通ユーティリティ（sample-api.mjs manifest を含む）
```

## スクリプトカテゴリ

### ドキュメント生成

|スクリプト|説明|実行元|
|---|---|---|
|`gen-portal-docs.mjs`|`manifest.yaml` に基づきソースドキュメントをポータルの `guides/` にコピー|`make gen-docs`|
|`gen-docs-json.mjs`|ポータルアプリ用のナビゲーション `docs.json` を生成|`make gen-docs`|
|`build-portal.mjs`|ポータルフロントエンド（`docs/portal/src/main.jsx`）を esbuild で `docs/portal/dist/` 配下（`bundle.js` / `bundle.css` + 遅延チャンク）へバンドルし、`mermaid.min.js` も同じく dist/ へ配置。従来の CDN + ブラウザ内 Babel 構成を置き換え。|`make gen-portal-build`|

### バージョニング

|スクリプト|説明|実行元|
|---|---|---|
|`semver.mjs`|セマンティックバージョンのバンプ（patch / minor / major）|リリースワークフロー|
|`stamp-openapi-version.mjs`|`release/vX.Y.Z` のブランチ名から `X.Y.Z` を導出し `openapi.yaml` の `info.version` に書き込む（先頭の `version:` 行のみ・冪等・非 release ref は no-op）。契約版のみで SHA / build metadata は付けない（commit 単位の追跡は runtime の `/version` の責務）。依存ゼロの ESM で、素の runner `node` で動く。|`auto-generate-docs.yaml`|
|`sync-versions/`|Go 実装の sync ツール。`mise.toml` の `[tools]` table を行ベース parser で解析し（外部依存ゼロ）、`go` / `node` / `python` を `go.mod` の `go` directive と `docker/*/Dockerfile` の `FROM golang:` / `FROM node:` / `FROM python:` 行へ反映する。version 存在・ファイル存在・期待マッチ数の事前 validate を全 rule で通してからファイル単位 atomic に書き出すため、partial state にならない。|`make sync-versions`|

その他のツールのバージョンは [`mise.toml`](../mise.toml) を SSOT として管理しています。各環境（host / docker / CI）は必要なものだけ `mise install <tool>` で個別に取得するため、sync スクリプトは不要です。

### Makefile サポート

|スクリプト|説明|実行元|
|---|---|---|
|`make_help.mjs`|`.makefiles/*.mk` を解析してターゲット説明を表示|`make help`|

### コード生成

|スクリプト|説明|実行元|
|---|---|---|
|`genctxkey/`|Echo コンテキストキーヘルパーの生成（Go コードジェネレータ）|`make gen-ctxkey`|

詳細は [genctxkey/README.ja.md](genctxkey/README.ja.md) を参照。

### 初期設定（`setup/`）

ボイラープレートから新規プロジェクトを作成する際の設定スクリプトです。

|スクリプト|説明|
|---|---|
|`replace-module.mjs`|Go モジュール名を全 `.go`、`go.mod` 等で置換|
|`replace-app-metadata.mjs`|env ファイルと OpenAPI 仕様のアプリ名・説明を置換|
|`replace-license-copyright.mjs`|LICENSE の著作権者名と年を置換|
|`replace-repository-reference.mjs`|README と OpenAPI の GitHub リポジトリ参照を置換|
|`remove-sample-api.mjs`|サンプルAPI(`user`/`product`/`order`)を削除。`lib/sample-api.mjs` に宣言したパスを削除し、共有 DI モジュールと `openapi.yaml` の `sample-api` マーカーブロックを除去する。再生成・整形・Lint まで行うには `make setup-remove-sample-api` 経由で実行する。|

すべての setup スクリプトはプレビュー用の `--dry-run` をサポートしています。

`remove-sample-api.mjs` の削除対象とマーカーは [`lib/sample-api.mjs`](setup/lib/sample-api.mjs) に宣言されています。サンプルは3ドメイン構成（`user` はフルスタック、`product`/`order` は拡張予定の DB スタブ）で、拡張時は該当ドメインブロックにパスを追記し、混在行を `// sample-api:begin … :end`（または `// sample-api:line`）で囲むだけで対象に含まれます。

## 注意点

- ドキュメント生成スクリプトは Node.js と `js-yaml` が必要（`docker/tools/` 経由でインストール）
- setup スクリプトは一度だけ使用 — ボイラープレートから新規プロジェクト作成時に実行
- AI エージェントは明示的な指示がない限りこのディレクトリを変更しないこと
