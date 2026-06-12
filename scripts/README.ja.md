# scripts

[English](README.md) | 日本語

`scripts/` には、コード生成・ドキュメント・バージョニング・プロジェクト初期設定のための**ユーティリティスクリプト**が格納されています。

## ディレクトリ構成

```text
scripts/
├── gen-docs-json.mjs           # ポータルナビゲーション用 docs.json の生成
├── gen-portal-docs.mjs         # manifest.yaml に基づくドキュメントのポータルへのコピー
├── semver.mjs                  # セマンティックバージョニングヘルパー（patch/minor/major）
├── sync-versions/              # mise.toml の go / node / python を go.mod と Dockerfile FROM へ反映（Go 実装）
├── make_help.mjs                # Make ターゲットのヘルプ出力生成
├── genctxkey/                  # コンテキストキーコードジェネレータ（Go）
└── setup/                     # プロジェクト初期設定スクリプト
    ├── replace-module.mjs
    ├── replace-app-metadata.mjs
    ├── replace-license-copyright.mjs
    ├── replace-repository-reference.mjs
    └── lib/                   # setup スクリプト共通ユーティリティ
```

## スクリプトカテゴリ

### ドキュメント生成

|スクリプト|説明|実行元|
|---|---|---|
|`gen-portal-docs.mjs`|`manifest.yaml` に基づきソースドキュメントをポータルの `guides/` にコピー|`make gen-docs`|
|`gen-docs-json.mjs`|ポータルアプリ用のナビゲーション `docs.json` を生成|`make gen-docs`|

### バージョニング

|スクリプト|説明|実行元|
|---|---|---|
|`semver.mjs`|セマンティックバージョンのバンプ（patch / minor / major）|リリースワークフロー|
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

すべての setup スクリプトはプレビュー用の `--dry-run` をサポートしています。

## 注意点

- ドキュメント生成スクリプトは Node.js と `js-yaml` が必要（`docker/tools/` 経由でインストール）
- setup スクリプトは一度だけ使用 — ボイラープレートから新規プロジェクト作成時に実行
- AI エージェントは明示的な指示がない限りこのディレクトリを変更しないこと
