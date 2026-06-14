# GitHub Actions Workflows

[English](README.md) | 日本語

このディレクトリには CI/CD 用の GitHub Actions ワークフロー定義を格納しています。ワークフローは目的別にグルーピングされており、PR ゲート（lint / test / セキュリティスキャン）、push 起点のデプロイ、リリースブランチ起点のドキュメント再生成という構成です。

## トリガー戦略

| グループ | 発火タイミング | 目的 |
| --- | --- | --- |
| CI チェック | 全 PR | lint / test / 生成物整合性が失敗したらマージブロック |
| セキュリティ | 全 PR + 週次スケジュール（Trivy / CodeQL）+ push ベースライン（CodeQL） | コード / 依存 / イメージ / Go ランタイムの脆弱性を surface |
| デプロイ | `production` / `staging` / `develop` への push | 成果物ビルド、マイグレーション実行、アプリ / docs portal をデプロイ |
| ドキュメント | `release/*` への push | OpenAPI / ER / portal ドキュメントを再生成し auto-sync PR を作成 |

## ワークフロー一覧

### CI チェック（Pull Request）

|ワークフロー|ファイル|説明|
|---|---|---|
|Go Lint|`lint.yaml`|golangci-lint による Go コードの静的解析|
|Go Test|`test.yaml`|Go テスト実行とカバレッジレポート|
|Module Tidy Check|`tidy-check.yaml`|go.mod / go.sum の整合性検証|
|SQL Lint|`sql-lint.yaml`|sqlfluff による migration / DML / seed SQL の検証|
|Actions Lint|`actions-lint.yaml`|actionlint による GitHub Actions 定義（ワークフロー / composite action）の検証（go_tool_runner 経由）|
|Migration Check|`migration-check.yaml`|マイグレーションファイルの検証（重複、欠番、up/down ペア）|
|Sync Versions Check|`sync-versions-check.yaml`|mise.toml のバージョンが go.mod / 各 Dockerfile / README へ伝播済みか検証|
|Generated Go Artifacts Check|`gen-go-artifacts-check.yaml`|生成済み Go コードとコミット済み成果物の一致検証|
|Generated Database Artifacts Check|`gen-db-artifacts-check.yaml`|生成済み sqlc コードとコミット済み成果物の一致検証|
|Generated OpenAPI Artifacts Check|`gen-oapi-artifacts-check.yaml`|OpenAPI バンドルとドキュメントの一致検証|
|App Boot Check|`app-di-startup-check.yaml`|DB 付きでアプリケーションサーバが正常に起動するか検証|
|Job Boot Check|`job-boot-check.yaml`|ジョブのエントリポイントが起動し、未知のジョブを拒否するか検証|

### セキュリティ

|ワークフロー|ファイル|説明|
|---|---|---|
|CodeQL Scan|`code-ql.yaml`|CodeQL によるセキュリティ脆弱性分析|
|Dependency Scan|`trivy-fs.yaml`|Trivy によるライブラリ脆弱性スキャン(開発者向け)|
|Release Dependency Scan|`trivy-release-gate.yaml`|develop/staging/production 向け PR での Trivy 依存スキャン|
|Image Scan|`image-scan.yaml`|Docker イメージビルド + SBOM 生成 + Trivy スキャン|
|Vulnerability Scan|`vulnerability-check.yaml`|govulncheck による Go パッケージ脆弱性検出|

### デプロイ（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Deploy App|`deploy-app.yaml`|production/staging/develop への push|Docker イメージのビルド・プッシュ、マイグレーション実行、デプロイ|
|Deploy Docs|`deploy-docs.yaml`|production への push（docs 変更時）|ドキュメントポータルを GitHub Pages にデプロイ|

### ドキュメント生成（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Auto-generate Docs|`auto-generate-docs.yaml`|release/* への push|OpenAPI ドキュメント、ER 図、ポータルドキュメントを自動生成|

## 共通 Composite Action

再利用可能な composite action は [`.github/actions/`](../actions/) に配置しています：

|アクション|目的|
|---|---|
|`setup-postgres`|Postgres サービスコンテナの待機・初期化（DB 依存ジョブで使用）|
|`upsert-pr-comment`|マーカーで既存コメントを検出して update / create する PR コメントの upsert。Commit / UpdatedAt フッターを共通付与し、結果コメント系ワークフローで使用|

## 補足

- `auto-generate-docs.yaml` は `auto/docs-update/<base>-<run-id>` というブランチ名で auto-PR を作成。再帰実行を避けるため自己ブランチでは workflow をスキップ
- デプロイ系 workflow の target ブランチ（`production` / `staging` / `develop`）はすべてブランチ保護を有効化。マージは必ず PR レビュー経由
- セキュリティスキャンは全 PR で実行（Trivy の FS / image と CodeQL は新規公表 CVE / クエリ検知のため週次 `schedule` でも実行。CodeQL は code scanning ベースライン維持のため `release/*` とデプロイ系ブランチへの push でも実行）。CodeQL / Trivy で high-severity が出るとブランチ保護ルールでマージブロック
- `auto-generate-docs.yaml` の `Detect changes` ステップはカバレッジ HTML / SchemaSpy のタイムスタンプ揺れを除外し、無意味な PR が発火しないよう設計
