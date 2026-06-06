# GitHub Actions Workflows

[English](README.md) | 日本語

このディレクトリには CI/CD 用の GitHub Actions ワークフロー定義を格納しています。ワークフローは目的別にグルーピングされており、PR ゲート（lint / test / セキュリティスキャン）、push 起点のデプロイ、リリースブランチ起点のドキュメント再生成という構成です。

## トリガー戦略

| グループ | 発火タイミング | 目的 |
| --- | --- | --- |
| CI チェック | 全 PR | lint / test / 生成物整合性が失敗したらマージブロック |
| セキュリティ | 全 PR（および default ブランチ push） | コード / 依存 / イメージ / Go ランタイムの脆弱性を surface |
| デプロイ | `production` / `staging` / `develop` への push | 成果物ビルド、マイグレーション実行、アプリ / docs portal をデプロイ |
| ドキュメント | `release/*` への push | OpenAPI / ER / portal ドキュメントを再生成し auto-sync PR を作成 |

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
|Application Boot|`app-di-startup-check.yaml`|DB 付きでアプリケーションが正常に起動するか検証|

### セキュリティ（Pull Request）

|ワークフロー|ファイル|説明|
|---|---|---|
|Code Security Scan|`code-ql.yaml`|CodeQL によるセキュリティ脆弱性分析|
|Dependency Vulnerability Scan|`trivy-fs.yaml`|Trivy によるライブラリ脆弱性スキャン(開発者向け)|
|Release Dependency Vulnerability Scan|`trivy-release-gate.yaml`|develop/staging/production 向け PR での Trivy 依存スキャン|
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

## 補足

- `auto-generate-docs.yaml` は `auto/docs-update/<base>-<run-id>` というブランチ名で auto-PR を作成。再帰実行を避けるため自己ブランチでは workflow をスキップ
- デプロイ系 workflow の target ブランチ（`production` / `staging` / `develop`）はすべてブランチ保護を有効化。マージは必ず PR レビュー経由
- セキュリティスキャンは全 PR で実行。CodeQL / Trivy で high-severity が出るとブランチ保護ルールでマージブロック
- `auto-generate-docs.yaml` の `Detect changes` ステップはカバレッジ HTML / SchemaSpy のタイムスタンプ揺れを除外し、無意味な PR が発火しないよう設計
