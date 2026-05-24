# scripts

[English](README.md) | 日本語

`scripts/` には、コード生成・ドキュメント・バージョニング・プロジェクト初期設定のための**ユーティリティスクリプト**が格納されています。

## ディレクトリ構成

```text
scripts/
├── gen-docs-json.cjs           # ポータルナビゲーション用 docs.json の生成
├── gen-portal-docs.cjs         # manifest.yaml に基づくドキュメントのポータルへのコピー
├── replace-tools-version.cjs   # tools.yaml からツールバージョンを置換
├── semver.cjs                  # セマンティックバージョニングヘルパー（patch/minor/major）
├── make_help.sh                # Make ターゲットのヘルプ出力生成
├── genctxkey/                  # コンテキストキーコードジェネレータ（Go）
└── setup/                     # プロジェクト初期設定スクリプト
    ├── replace-module.cjs
    ├── replace-app-metadata.cjs
    ├── replace-license-copyright.cjs
    ├── replace-repository-reference.cjs
    ├── remove-debug-handlers.cjs
    └── lib/                   # setup スクリプト共通ユーティリティ
```

## スクリプトカテゴリ

### ドキュメント生成

|スクリプト|説明|実行元|
|---|---|---|
|`gen-portal-docs.cjs`|`manifest.yaml` に基づきソースドキュメントをポータルの `guides/` にコピー|`make gen-docs`|
|`gen-docs-json.cjs`|ポータルアプリ用のナビゲーション `docs.json` を生成|`make gen-docs`|

### バージョニングとツール

|スクリプト|説明|実行元|
|---|---|---|
|`replace-tools-version.cjs`|`tools.yaml` から Dockerfile / 設定ファイルのツールバージョンを置換|`make replace-versions`|
|`semver.cjs`|セマンティックバージョンのバンプ（patch / minor / major）|リリースワークフロー|

### Makefile サポート

|スクリプト|説明|実行元|
|---|---|---|
|`make_help.sh`|`.makefiles/*.mk` を解析してターゲット説明を表示|`make help`|

### コード生成

|スクリプト|説明|実行元|
|---|---|---|
|`genctxkey/`|Echo コンテキストキーヘルパーの生成（Go コードジェネレータ）|`make gen-ctxkey`|

詳細は [genctxkey/README.ja.md](genctxkey/README.ja.md) を参照。

### 初期設定（`setup/`）

ボイラープレートから新規プロジェクトを作成する際の設定スクリプトです。

|スクリプト|説明|
|---|---|
|`replace-module.cjs`|Go モジュール名を全 `.go`、`go.mod` 等で置換|
|`replace-app-metadata.cjs`|env ファイルと OpenAPI 仕様のアプリ名・説明を置換|
|`replace-license-copyright.cjs`|LICENSE の著作権者名と年を置換|
|`replace-repository-reference.cjs`|README と OpenAPI の GitHub リポジトリ参照を置換|
|`remove-debug-handlers.cjs`|デバッグエンドポイントを削除（handler、OpenAPI paths / requests / responses）|

すべての setup スクリプトはプレビュー用の `--dry-run` をサポートしています。

## 注意点

- ドキュメント生成スクリプトは Node.js と `js-yaml` が必要（`docker/tools/` 経由でインストール）
- setup スクリプトは一度だけ使用 — ボイラープレートから新規プロジェクト作成時に実行
- AI エージェントは明示的な指示がない限りこのディレクトリを変更しないこと
