# GitHub Actions Workflows

[English](README.md) | 日本語

このディレクトリには CI/CD 用の GitHub Actions ワークフロー定義を格納しています。ワークフローは目的別にグルーピングされており、PR ゲート（lint / test / セキュリティスキャン）、push 起点のデプロイ、リリースブランチ起点のドキュメント再生成という構成です。

## トリガー戦略

| グループ | 発火タイミング | 目的 |
| --- | --- | --- |
| CI チェック | 全 PR | lint / test / 生成物整合性が失敗したらマージブロック |
| セキュリティ | ツールごとのマトリクス（後述） | コード / 依存 / イメージ / ワークフロー定義 / コミット済みシークレットの問題を surface |
| デプロイ | `production` / `staging` / `develop` への push | 成果物ビルド、マイグレーション実行、アプリ / docs portal をデプロイ |
| ドキュメント | `release/*` への push | OpenAPI / ER / portal ドキュメントを再生成し auto-sync PR を作成 |

## ワークフロー一覧

### CI チェック（Pull Request）

|ワークフロー|ファイル|説明|
|---|---|---|
|Go Lint|`go-lint.yaml`|golangci-lint による Go コードの静的解析|
|Go Test|`go-test.yaml`|Go テスト実行とカバレッジレポート|
|Module Tidy Check|`tidy-check.yaml`|go.mod / go.sum の整合性検証|
|SQL Lint|`sql-lint.yaml`|sqlfluff による migration / DML / seed SQL の検証|
|Actions Lint|`actions-lint.yaml`|actionlint による GitHub Actions 定義（ワークフロー / composite action）の検証（go_tool_runner 経由）|
|Migration Check|`migration-check.yaml`|マイグレーションファイルの検証（重複、欠番、up/down ペア）|
|Sync Versions Check|`sync-versions-check.yaml`|mise.toml のバージョンが go.mod / 各 Dockerfile / README へ伝播済みか検証|
|Generated Go Artifacts Check|`gen-go-artifacts-check.yaml`|生成済み Go コードとコミット済み成果物の一致検証|
|Generated Database Artifacts Check|`gen-db-artifacts-check.yaml`|生成済み sqlc コードとコミット済み成果物の一致検証|
|Generated OpenAPI Artifacts Check|`gen-oapi-artifacts-check.yaml`|OpenAPI バンドルとドキュメントの一致検証|
|OpenAPI Lint|`oapi-lint.yaml`|OpenAPI 定義を `redocly lint` で検証（命名 / casing / description / 未使用コンポーネント）|
|App Boot Check|`app-di-startup-check.yaml`|DB 付きでアプリケーションサーバが正常に起動するか検証|
|Job Boot Check|`job-boot-check.yaml`|ジョブのエントリポイントが起動し、未知のジョブを拒否するか検証|
|Worker Boot Check|`worker-boot-check.yaml`|worker のエントリポイントが起動（DI / DB）し、未知の worker を拒否するか検証|
|Dockerfile Lint|`docker-lint.yaml`|hadolint による Dockerfile の検証（go_tool_runner 経由）|
|Pin Actions Check|`pin-actions-check.yaml`|GitHub Actions が SHA でピン留めされているか検証（サプライチェーン対策）|
|Pin Images Check|`pin-images-check.yaml`|Docker base image が lockfile 通り digest でピン留めされているか検証（サプライチェーン対策）|

### セキュリティ

|ワークフロー|ファイル|説明|
|---|---|---|
|CodeQL Scan|`code-ql.yaml`|CodeQL によるセキュリティ脆弱性分析|
|Dependency Scan|`trivy-fs.yaml`|Trivy によるライブラリ脆弱性スキャン(開発者向け)|
|Release Dependency Scan|`trivy-release-gate.yaml`|develop/staging/production 向け PR での Trivy 依存スキャン|
|Image Scan|`image-scan.yaml`|Docker イメージビルド + SBOM 生成 + Trivy スキャン|
|Vulnerability Scan|`vulnerability-check.yaml`|govulncheck による Go パッケージ脆弱性検出|
|OSV Scan|`osv-scanner.yaml`|OSV データベースによる Go モジュール / npm lockfile 横断の脆弱性スキャン|
|Secret Scan|`secret-scan.yaml`|gitleaks によるコミット済みシークレットの検出（go_tool_runner 経由）|
|Secret Scan (TruffleHog)|`trufflehog.yaml`|TruffleHog による**検証済み**シークレット（実際に有効なクレデンシャル）の検出|
|Actions Static Analysis|`zizmor.yaml`|zizmor によるワークフロー / composite action 定義自体の静的解析|
|Dependency Review|`dependency-review.yaml`|PR が新たに持ち込む脆弱な依存をマージ前にブロック|
|OpenSSF Scorecard|`scorecard.yaml`|リポジトリのセキュリティ姿勢のスコアリングと結果の公開|

各スキャナは可能な限り SARIF を GitHub code scanning へ送り、結果は共通の `upsert-pr-comment` アクションで PR にコメントします。

#### セキュリティのトリガーマトリクス

各ツールは「結果が実際に変わりうる場所」で走らせています。PR はその変更自身が持ち込むリスクを surface し、protected branch への push はブランチ保護が判断材料にする code scanning のベースラインを残し、定期実行は「コードが変わらなくても結果が変わる」種別（新規公表 CVE / 新規クエリ）にだけ設けます。

| 種別 | PR | protected branch への push | 定期 |
| --- | --- | --- | --- |
| gitleaks | 全 PR | 不要 | 不要 |
| TruffleHog | 全 PR の差分 | 不要 | 週次で履歴全体 |
| zizmor | Actions 関連ファイル変更時 | `develop` / `staging` / `production` / `release/*` | 不要 |
| Dependency Review | 依存関係変更 PR | 不要 | 不要 |
| govulncheck | Go・依存変更 PR | 同上 | 週次 |
| Trivy FS | Go・依存変更 PR | 同上 | 週次 |
| OSV-Scanner | 依存関係変更 PR | 同上 | 週次 |
| CodeQL | Go・依存変更 PR | 同上 | 週次 |
| OpenSSF Scorecard | 不要 | 既定ブランチのみ | 週次 |
| Image Scan | デプロイ先ブランチへの PR | 不要 | 週次 |

週次実行は月曜内でずらしています（`0 0` Trivy FS / Image Scan、`0 1` govulncheck、`0 2` TruffleHog、`0 3` OSV-Scanner、`0 4` Scorecard）。同一時刻に全スキャナが並ぶのを避けるためです。

#### ランナーのハードニング

このディレクトリの全ジョブは `step-security/harden-runner` を `egress-policy: audit` で先頭に置いています。ランナーの外向き通信とファイル改変を記録することで、侵害されたアクションやツールの推移的ダウンロードが可視化されます。`audit` は記録のみで、`block` へ移行するには許可エンドポイントの確定が前提になるため、監査データが溜まるまで意図的に見送っています。

### デプロイ（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Deploy App|`deploy-app.yaml`|production/staging/develop への push|Docker イメージのビルド・プッシュ（cosign による image 署名 + provenance / SBOM attestation）、マイグレーション実行、デプロイ|
|Deploy Docs|`deploy-docs.yaml`|production への push（docs 変更時）|ドキュメントポータルを GitHub Pages にデプロイ|

### ドキュメント生成（Push）

|ワークフロー|ファイル|トリガー|説明|
|---|---|---|---|
|Auto-generate Docs|`auto-generate-docs.yaml`|release/* への push|`release/vX.Y.Z` のブランチ名から OpenAPI `info.version` を同期し、OpenAPI バンドル / 埋め込み spec / ドキュメント、ER 図、ポータルドキュメントを自動生成|

## 共通 Composite Action

再利用可能な composite action は [`.github/actions/`](../actions/) に配置しています：

|アクション|目的|
|---|---|
|`setup-postgres`|Postgres サービスコンテナの待機・初期化（DB 依存ジョブで使用）|
|`upsert-pr-comment`|マーカーで既存コメントを検出して update / create する PR コメントの upsert。Commit / UpdatedAt フッターを共通付与し、結果コメント系ワークフローで使用|

## 補足

- `auto-generate-docs.yaml` は `auto/docs-update/<base>` というブランチ名で auto-PR を作成（release base ごとに 1 ブランチを `delete-branch: true` で再利用）。再帰実行を避けるため自己ブランチでは workflow をスキップ
- デプロイ系 workflow の target ブランチ（`production` / `staging` / `develop`）はすべてブランチ保護を有効化。マージは必ず PR レビュー経由
- セキュリティスキャンのトリガーは上記「セキュリティのトリガーマトリクス」でツールごとに定義。CodeQL / Trivy で high-severity が出るとブランチ保護ルールでマージブロック
- `trivy-fs.yaml` と `osv-scanner.yaml` は**修正版が存在する脆弱性でのみ**ゲートする。修正版のない advisory も code scanning と PR コメントには載るが、チェックを恒久的に赤くはしない。厳格版はデプロイ先ブランチ向け PR を見る `trivy-release-gate.yaml` が担当
- `trufflehog.yaml` は**検証済み**シークレットのみを報告し、生のシークレット値をジョブログ / PR コメント / artifact のいずれにも出さない。正規表現ベースの検知は `--redact` 付きの gitleaks が担当
- zizmor の例外設定は `.github/zizmor.yml`。`ignore` はファイル単位であり、同じ audit を踏む新規ワークフローは意図どおり落ちる。恒久的な allowlist ではなく、元の指摘を直したらエントリを消す運用
- `auto-generate-docs.yaml` の `Detect changes` ステップはカバレッジ HTML / SchemaSpy のタイムスタンプ揺れを除外し、無意味な PR が発火しないよう設計
