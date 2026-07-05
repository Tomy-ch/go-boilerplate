# アーキテクチャ決定記録（ADR）

English: [README.md](../../adr/README.md)

このディレクトリはプロジェクトの **アーキテクチャ上の決定** を保持する。1 ファイル = 1 つの不変な記録、[MADR-lite](https://adr.github.io/madr/) 形式。

ADR は 1 時点における単一の決定を記録する: コンテキスト、検討した選択肢、選んだもの、その影響。決定を上書きするとは ADR を編集することを **意味しない** — `Status` が `accepted` の *新しい* ADR を追加し、古い ADR を `superseded` にマークすることを意味する。*なぜかつて X を選んだのか* という記録は保存される。

## 何が対象か（そして何が対象でないか）

| 種別 | 例 | 置き場所 |
| --- | --- | --- |
| **decision** — 持続的な影響を持つ選択肢間の選択 | 「オニオンアーキテクチャを採用する」 | このディレクトリ（ADR） |
| **exclusion** — 意図的に「X をしない」という決定 | 「アプリ内レートリミッターなし」 | このディレクトリ（ADR） |
| **rule** — 日々強制される制約 / 決定の帰結 | 「コントローラーはインフラストラクチャをインポートしてはならない」 | `docs/rules.md`（ADR へのリンクを付ける場合がある） |
| **inventory** — コードと共に変化するカタログ | 直接依存関係テーブル | `docs/reference/dependencies.md`（生きたドキュメント） |

依存関係インベントリは ADR **ではない**: `go.mod` を追跡して継続的に変化し、不変の記録とは正反対である。依存関係を選定する *ポリシー* は決定（ADR）であり、その *一覧* は生きたリファレンスである。

## 規約

- **ファイル名**: `NNNN-kebab-title.md`、ゼロパディング 4 桁、単調増加。番号は上書き後も再利用されない。
- **順序**: 番号は依存関係 / 基礎的な順序に従う（原則 → コントラクト → レイヤー → サブシステム → 横断関心事 → exclusion）、発見順ではない。
- **ステータスライフサイクル**: `proposed` → `accepted` → (`superseded` | `deprecated`)。
- **不変**: `accepted` になったら、`Status` 行の編集と `Superseded-by` リンクの追加のみ。それ以外は書かれた通りのまま。
- **テンプレート**: [`template.ja.md`](template.ja.md) をコピーする。
- **メタ**: [`0000-record-architecture-decisions.ja.md`](0000-record-architecture-decisions.ja.md) は ADR の使用とこの分類の決定を記録する。
- **翻訳**: 各 ADR は `docs/ja/adr/` にミラーされる（`canonicalize-doc` フローを経由）。
- **Exclusion ADR**（意図的な「X はしない」）は `setup-review` タグを持ち、リポジトリセットアップフローがそれを列挙できるようにする。初期セットアップ時、フォークはこれらを **直接編集して** 自分のベースラインを確立してよい; 新 ADR による上書きモデルはその後の変更にのみ適用される。`docs/get-started/setup-repository.md` Phase 10 を参照のこと。

## ログ

`docs/decisions.md` およびリポジトリ全体の潜在的な決定から、すべての決定が ADR として具体化された。番号付けは依存関係 / 基礎的な順序に従う（原則 → コントラクト → HTTP → 永続化 → DI/設定 → 非同期サブシステム → 可観測性 → ツールチェーン/CI → プロセス → バイナリ/デプロイ → exclusion）。Exclusion ADR（意図的な「X はしない」）は `setup-review` タグ付き。

| # | 決定 | ステータス |
| --- | --- | --- |
| [0000](0000-record-architecture-decisions.ja.md) | アーキテクチャ上の決定を ADR として記録する | accepted |
| [0001](0001-avoid-lock-in.ja.md) | ロックイン回避を設計原則として採用する | accepted |
| [0002](0002-onion-architecture.ja.md) | 実用的なオニオンアーキテクチャを採用する | accepted |
| [0003](0003-interface-based-decoupling.ja.md) | 疎結合のためにインターフェースで境界を定義する（DIP） | accepted |
| [0004](0004-modular-monolith.ja.md) | モジュラーモノリスを採用する（マイクロサービスは非目標） | accepted |
| [0005](0005-driving-adapters-not-split-axis.ja.md) | REST / Worker / Job はドライビングアダプター、サービス分割の軸ではない | accepted |
| [0006](0006-structural-safety-via-tooling.ja.md) | ツールと CI で構造的安全性を強制する（depguard） | accepted |
| [0007](0007-agents-md-operational-contract.ja.md) | AI協働開発 — AGENTS.md を運用契約とする | accepted |
| [0008](0008-docs-as-canonical-source.ja.md) | ドキュメントを正典ソースとする戦略（英語正典 + ja ミラー + ポータル） | accepted |
| [0009](0009-openapi-first.ja.md) | API 契約を OpenAPI ファーストで定義する | accepted |
| [0010](0010-redocly-modular-spec-pipeline.ja.md) | 仕様をモジュラーな Redocly ファイルで作成し、バンドルしてから生成する | accepted |
| [0011](0011-oapi-codegen-strict-server.ja.md) | oapi-codegen の strict-server モードでタグ/ハンドラーごとに生成する | accepted |
| [0012](0012-retain-generated-openapi.ja.md) | バンドル済み openapi.gen.yaml をコミット済みのクロスリポジトリ契約アーティファクトとして保持する | accepted |
| [0013](0013-spec-driven-request-validation.ja.md) | リクエスト検証と認証を実行時に仕様から強制する。レスポンスは検証しない | accepted |
| [0014](0014-validation-value-authority.ja.md) | バリデーションのビジネス有効性における唯一の権威をドメイン層に定める | accepted |
| [0015](0015-boundary-value-ownership.ja.md) | OpenAPI はワイヤー契約であってドメインルールではない。リクエストはドメインのサブセット、ドメインはレスポンスのサブセット | accepted |
| [0016](0016-metrics-endpoint-auth-exception.ja.md) | /metrics は認証例外 — OpenAPI 検証の外に置き、独立した BasicAuth ミドルウェアで保護する | accepted |
| [0017](0017-echo-http-framework.ja.md) | HTTP フレームワークとして Echo を採用する | accepted |
| [0018](0018-priority-ordered-middleware-chain.ja.md) | ミドルウェアチェーンを優先順位付きのデータ駆動リストとして構築する | accepted |
| [0019](0019-outbound-http-resilience.ja.md) | アウトバウンドHTTPレジリエンス基盤の提供（リトライ / サーキットブレーカー / リトライバジェット / デュアルタイムアウト） | accepted |
| [0020](0020-egress-ssrf-guard.ja.md) | アウトバウンドHTTPに対するエグレスSSRF / ダイヤルガードセキュリティポスチャの採用 | accepted |
| [0021](0021-sql-first-data-access.ja.md) | SQLファーストのデータアクセス | accepted |
| [0022](0022-sqlc-type-safe-sql.ja.md) | sqlcによる型安全なSQLアクセスの生成 | accepted |
| [0023](0023-merged-dml-schema-as-sqlc-input.ja.md) | マージされたDMLおよびダンプされたスキーマをsqlcの単一入力として使用する | accepted |
| [0024](0024-append-only-immutable-migrations.ja.md) | マイグレーションを追記専用かつイミュータブルとして扱う | accepted |
| [0025](0025-sequential-migration-ids.ja.md) | CIで強制するギャップ・ペアチェックを伴う6桁連番マイグレーションIDの使用 | accepted |
| [0026](0026-master-data-via-migration.ja.md) | マスターデータをマイグレーション経由で投入する。トランザクショナルシードを本番から除外する | accepted |
| [0027](0027-lightweight-cqrs.ja.md) | 軽量CQRSの採用 — 書き込みにRepository、読み込みにQueryService | accepted |
| [0028](0028-system-cqrs-dml-category.ja.md) | CQRSの分割の外に位置する第4のDMLカテゴリとしてsystem_cqrsを導入する | accepted |
| [0029](0029-transaction-retry-idempotent-callers.ja.md) | シリアライゼーション競合時はトランザクションをリトライする。呼び出し元は冪等性を保証しなければならない | accepted |
| [0030](0030-uuidv7-identifiers.ja.md) | すべてのエンティティ主キーに UUIDv7（時刻順）識別子を使用する | accepted |
| [0031](0031-uber-fx-di.ja.md) | 依存性注入とライフサイクル管理に Uber Fx を採用する | accepted |
| [0032](0032-fx-neutral-di-abstraction.ja.md) | ニュートラルな DI 抽象（Registrar / Shutdowner）の背後に fx を封じ込める | accepted |
| [0033](0033-env-gated-wiring.ja.md) | DI を通じて環境ごとに実装を切り替える（環境ゲート結線） | accepted |
| [0034](0034-subsystem-typed-config-loaders.ja.md) | サブシステムスコープの envPrefix 型付き設定ローダー | accepted |
| [0035](0035-config-default-vs-required-governance.ja.md) | ガバナンス: コードデフォルト（不変）対ファイル必須（可変） | accepted |
| [0036](0036-immutable-fail-fast-config.ja.md) | 設定は不変、起動時に 1 回だけロード、フェイルファスト | accepted |
| [0037](0037-embedded-self-contained-binary.ja.md) | go:embed で設定（.env）とマイグレーションをバンドルし、自己完結型バイナリを実現する | accepted |
| [0038](0038-apperror-protocol-agnostic-errors.ja.md) | プロトコル非依存の集約エラー分類 (apperror) | accepted |
| [0039](0039-broker-agnostic-worker-scaffold.ja.md) | ブローカー非依存のプル・アック型ワーカースキャフォールド | accepted |
| [0040](0040-out-of-scope-push-streaming-brokers.ja.md) | プッシュ型ブローカーとストリーミングログ基盤はワーカーポートのスコープ外 | accepted (exclusion) |
| [0041](0041-sqs-adapter-opt-in.ja.md) | SQS アダプターはオプトインであり、デフォルトバイナリにリンクしない | accepted |
| [0042](0042-transactional-outbox.ja.md) | トランザクショナルアウトボックス — ビジネストランザクション内でイベントを発行する | accepted |
| [0043](0043-at-least-once-outbox-poll.ja.md) | ポーリングによる少なくとも1回のデリバリー（トランスポートレベルのリトライを無効化） | accepted |
| [0044](0044-skip-locked-outbox-relay.ja.md) | SELECT FOR UPDATE SKIP LOCKED を使った単一トランザクションリレー（複数インスタンス間で安全） | accepted |
| [0045](0045-message-id-idempotency-propagation.ja.md) | アウトボックスの message_id をレシーバーの Idempotency-Key として伝播する | accepted |
| [0046](0046-outbox-dead-after-max-attempts.ja.md) | MaxAttempts = 10 到達でメッセージをデッド状態にする（手動リプレイまで終端） | accepted |
| [0047](0047-outbox-retention-gc.ja.md) | 発行済み行の 7 日間保持 GC（10,000 件単位のバッチ） | accepted |
| [0048](0048-publisher-http-profile-isolation.ja.md) | パブリッシャーの非標準 HTTP プロファイルをリレー内に隔離する | accepted |
| [0049](0049-relay-resident-gc-oneshot.ja.md) | リレーは常駐プロセス、GC はワンショット cron ジョブ | accepted |
| [0050](0050-single-tx-at-most-once-idempotency.ja.md) | claim・ビジネス関数・complete を単一トランザクションで実行してアットモストワンスを保証する | accepted |
| [0051](0051-idempotency-scope-required.ja.md) | クロスユーザーのキー衝突を防ぐためすべての Store 呼び出しに明示的スコープを必須とする | accepted |
| [0052](0052-idempotency-fixed-ttl.ja.md) | 冪等性キーの TTL を 24 時間に固定しルート別設定を設けない | accepted |
| [0053](0053-idempotency-response-persistence.ja.md) | 決定論的リプレイを可能にするためレスポンスボディを JSON で永続化する（PII トレードオフを許容） | accepted |
| [0054](0054-idempotency-gc-separate-job.ja.md) | 冪等性キーのガベージコレクションを独立したワンショット CLI ジョブとして実行する | accepted |
| [0055](0055-idempotency-orthogonal-concerns.ja.md) | 冪等性をオプティミスティックロックおよびレート制限と直交に保つ | accepted (exclusion) |
| [0056](0056-job-fresh-fx-app-per-run.ja.md) | ジョブ起動ごとに新しい fx.App を構築する（ワンショットライフサイクル） | accepted |
| [0057](0057-job-no-worker-machinery.ja.md) | ジョブにはブローカー・サーキットブレーカー・ドレイン・ヘルス機構を意図的に設けない | accepted (exclusion) |
| [0058](0058-job-explicit-registration.ja.md) | Job は明示的に登録する（自動検出なし） | accepted |
| [0059](0059-config-driven-observability-gating.ja.md) | 設定駆動によるオブザーバビリティゲーティング | accepted |
| [0060](0060-vendor-neutral-otlp-export.ja.md) | ベンダー中立の OTLP 専用エクスポート（バックエンドは Collector に委譲） | accepted |
| [0061](0061-official-otel-semconv.ja.md) | 公式 OpenTelemetry セマンティック規約のみを使用し、カスタム semconv の発明や型付き設定へのベンダーキー追加は行わない | accepted (exclusion) |
| [0062](0062-dual-path-metrics.ja.md) | メトリクスは 2 経路を通る — OTLP プッシュと Prometheus スクレイプ | accepted |
| [0063](0063-lifecycle-independent-provider.ja.md) | オブザーバビリティプロバイダーはライフサイクル非依存（ProviderShutdowner） | accepted |
| [0064](0064-fixed-default-sampling.ja.md) | SDK デフォルトサンプリングを固定し、サンプリングを環境変数ノブとして公開しない | accepted (exclusion) |
| [0065](0065-library-selection-policy.ja.md) | 単一責任のライブラリ選定ポリシー | accepted |
| [0066](0066-bridge-instrumentation-exceptions.ja.md) | ブリッジ / 計装ライブラリを有界な SRP 例外として認める | accepted |
| [0067](0067-containerized-pinned-toolchain.ja.md) | 再現性のために mise でバージョン固定されたコンテナ化ツールチェーンを使用する | accepted |
| [0068](0068-mise-ssot-drift-gate.ja.md) | mise.toml を単一の情報源とし、バージョンを下流に伝播させ CI でドリフトを検知する | accepted |
| [0069](0069-make-single-entrypoint.ja.md) | Make を単一のツールエントリポイントとし、.mk 登録とセルフドキュメンティングなヘルプを提供する | accepted |
| [0070](0070-scripts-in-node-go.ja.md) | 運用スクリプトは scripts/ に Node（.mjs）または Go で配置し、シェルスクリプトは使用しない | accepted |
| [0071](0071-docker-compose-dev-environment.ja.md) | ローカル開発環境はプロファイルで分離されたサービスを持つ Docker Compose で提供する | accepted |
| [0072](0072-two-layer-golangci-config.ja.md) | 2 層の golangci 設定——最小デフォルトと完全な権威ゲート | accepted |
| [0073](0073-local-hooks-mirror-ci.ja.md) | ローカル git フックは CI 契約を複製する（local == CI、グロブスコープ、バイパス後に一度検証） | accepted |
| [0074](0074-coverage-hard-gate.ja.md) | 総カバレッジ 90% を CI のハードゲートとし、例外ガバナンスパスを設ける | accepted |
| [0075](0075-ci-real-graph-boot-check.ja.md) | CI は実際の Postgres に対して実際の fx グラフを起動する（スタートアップ検証） | accepted |
| [0076](0076-generated-artifact-drift-gate.ja.md) | 生成成果物ドリフトゲートとリリースブランチ集約型自動生成ボット | accepted |
| [0077](0077-multi-layer-security-scanning.ja.md) | 多層セキュリティスキャン（到達可能性フィルタ付き govulncheck + スケジュール CodeQL SAST + シークレット + FS スキャン） | accepted |
| [0078](0078-sha-pinned-actions.ja.md) | GitHub Actions を SHA でピン留めし、サプライチェーン隔離を適用する | accepted |
| [0079](0079-rollback-integration-tests.ja.md) | インフラ統合テストはリアル DB に対してセンチネルエラーロールバックで実行する | accepted |
| [0080](0080-multi-model-adversarial-review.ja.md) | ファインダー・ベリファイアーサブエージェントによるマルチモデル敵対的レビューを使用する | accepted |
| [0081](0081-lean-a-spec-scaffold.ja.md) | スペックファイルからドメインとユースケースのみスキャフォールドし、コントローラーとインフラは生成コードから導出する | accepted |
| [0082](0082-cli-humble-object-split.ja.md) | CLI ハンブルオブジェクト分割（薄い cmd/ シェル + テスト可能な internal/cli コア） | accepted |
| [0083](0083-single-multi-command-binary.ja.md) | すべてのロールを 1 つのマルチコマンドバイナリに集約する | accepted |
| [0084](0084-single-runtime-image.ja.md) | コマンドオーバーライドによる単一ランタイムイメージ（目的別イメージなし） | accepted |
| [0085](0085-hardened-alpine-runtime.ja.md) | ハードニング Alpine をランタイムベースとして使用し、distroless/scratch は使用しない | accepted (exclusion) |
| [0086](0086-per-environment-images.ja.md) | 環境別イメージ（.env マトリックス × APP_ENV ビルド引数、ビルド時に固定） | accepted |
| [0087](0087-predeploy-oneshot-migration.ja.md) | マイグレーションはデプロイ前のワンショットとして実行し、アプリケーション起動時の自動マイグレーションは行わない | accepted (exclusion) |
| [0088](0088-release-image-supply-chain.ja.md) | リリースイメージのサプライチェーン完全性（cosign 署名 + プロベナンス + SBOM） | accepted |
| [0089](0089-vendor-neutral-deploy-skeleton.ja.md) | デプロイはベンダー中立のスケルトン（ビルド/署名は実装済み；クラウド CD はテンプレート；レジストリは固定しない） | accepted |
| [0090](0090-docs-via-github-pages.ja.md) | docs/ の静的コンテンツを GitHub Pages で公開（production プッシュ時にリリース） | accepted |
| [0091](0091-no-in-app-rate-limiter.ja.md) | アプリケーション内レートリミッターを提供しない | accepted (exclusion) |
| [0092](0092-scheduled-job-concurrency-delegated.ja.md) | スケジュールジョブの同時実行制御をアプリ内で行わず、スケジューラに委譲する | accepted (exclusion) |
| [0093](0093-no-generic-cache-abstraction.ja.md) | 汎用 Cache 抽象化を提供しない | accepted (exclusion) |

フロントマターフィールド: `status`、`date`、`deciders`、`supersedes` / `superseded-by`、`tags`。
Consequences は MADR 標準に従う（`Positive` / `Negative`; 任意で `Neutral`）。
