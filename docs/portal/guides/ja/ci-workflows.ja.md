# GitHub Actions Workflows

[English](README.md) | 日本語

このディレクトリには CI/CD 用の GitHub Actions ワークフロー定義を格納しています。

## ワークフロー一覧

### CI チェック（Pull Request）

|ワークフロー|ファイル|説明|
|---|---|---|
|Golang Lint|`lint.yaml`|golangci-lint による Go コードの静的解析|
|Golang Test|`test.yaml`|Go テスト実行とカバレッジレポート|
|Go Module Consistency|`tidy-check.yaml`|go.mod / go.sum の整合性検証|
|SQL Lint|`sql-lint.yaml`|sqlfluff による migration / DML / seed SQL の検証|
|Migration Check|`migration-check.yaml`|マイグレーションファイルの検証（重複、欠番、up/down ペア）|
|Generated Go Artifacts|`gen-go-artifacts-check.yaml`|生成済み Go コードとコミット済み成果物の一致検証|
|Generated DB Artifacts|`gen-db-artifacts-check.yaml`|生成済み sqlc コードとコミット済み成果物の一致検証|
|Generated OpenAPI Artifacts|`gen-oapi-artifacts-check.yaml`|OpenAPI バンドルとドキュメントの一致検証|
|Generator Versions|`gen-versions-check.yaml`|生成ツールのバージョン同期検証|
|Application Boot|`app-di-startup-check.yaml`|DB 付きでアプリケーションが正常に起動するか検証|

### セキュリティ（Pull Request）

|ワークフロー|ファイル|説明|
|---|---|---|
|Code Security Scan|`code-ql.yaml`|CodeQL によるセキュリティ脆弱性分析|
|Dependency Vulnerability Scan|`trivy-fs.yaml`|Trivy による OS / ライブラリ脆弱性スキャン|
|Docker Image Scan|`image-scan.yaml`|Docker イメージビルド + SBOM 生成 + Trivy スキャン|
|Go Vulnerability Analysis|`vulnerability-check.yaml`|govulncheck による Go パッケージ脆弱性検出|

### デプロイ（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Application Deployment|`deploy-app.yaml`|production/staging/develop への push|Docker イメージのビルド・プッシュ、マイグレーション実行、デプロイ|
|Deploy Docs Portal|`deploy-docs.yaml`|production への push（docs 変更時）|ドキュメントポータルを GitHub Pages にデプロイ|

### ドキュメント生成（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Auto-generate Docs PR|`auto-generate-docs.yaml`|release/* への push|OpenAPI ドキュメント、ER 図、ポータルドキュメントを自動生成|
