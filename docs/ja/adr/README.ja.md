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
| [0007](0007-agents-md-operational-contract.ja.md) | AI 協働開発 — AGENTS.md を運用コントラクトとする | accepted |
| [0008](0008-docs-as-canonical-source.ja.md) | Docs-as-canonical-source 戦略（英語正典 + ja ミラー + ポータル） | accepted |
| [0009](0009-openapi-first.ja.md) | API コントラクトを OpenAPI ファーストで定義する | accepted |
| [0010](0010-redocly-modular-spec-pipeline.ja.md) | 仕様をモジュール化した Redocly ファイルで記述し、バンドルしてから生成する | accepted |
| [0011](0011-oapi-codegen-strict-server.ja.md) | oapi-codegen でタグ/ハンドラーごとに strict-server モードで生成する | accepted |
| [0012](0012-spec-driven-request-validation.ja.md) | 実行時にスペックからリクエストを検証し auth を強制する; レスポンスは検証しない | accepted |
| [0013](0013-boundary-value-ownership.ja.md) | OpenAPI はワイヤーコントラクトであり、ドメインルールではない; リクエストはドメインのサブセット、ドメインはレスポンスのサブセット | accepted |
| [0014](0014-metrics-endpoint-auth-exception.ja.md) | /metrics は認証例外 — OpenAPI バリデーション外、別途 BasicAuth ミドルウェアで保護 | accepted |
| [0015](0015-echo-http-framework.ja.md) | Echo を HTTP フレームワークとして採用する | accepted |
| [0016](0016-priority-ordered-middleware-chain.ja.md) | ミドルウェアチェーンを優先度順のデータ駆動リストとして構築する | accepted |
| [0017](0017-outbound-http-resilience.ja.md) | アウトバウンド HTTP レジリエンス基盤を提供する（リトライ / サーキットブレーカー / リトライバジェット / デュアルタイムアウト） | accepted |
| [0018](0018-egress-ssrf-guard.ja.md) | アウトバウンド HTTP に egress SSRF / ダイアルガード セキュリティ姿勢を採用する | accepted |
| [0019](0019-sql-first-data-access.ja.md) | SQL ファーストのデータアクセス | accepted |
| [0020](0020-sqlc-type-safe-sql.ja.md) | sqlc で型安全な SQL アクセスを生成する | accepted |
| [0021](0021-merged-dml-schema-as-sqlc-input.ja.md) | マージ済み DML とダンプスキーマを sqlc の単一入力として使用する | accepted |
| [0022](0022-append-only-immutable-migrations.ja.md) | マイグレーションを追記のみ・不変として扱う | accepted |
| [0023](0023-sequential-migration-ids.ja.md) | CI で強制するギャップ・ペアチェック付きの 6 桁連番マイグレーション ID を使用する | accepted |
| [0024](0024-master-data-via-migration.ja.md) | マスターデータをマイグレーションで提供する; トランザクショナルシードを本番から除外する | accepted |
| [0025](0025-lightweight-cqrs.ja.md) | 軽量 CQRS を採用する — 書き込みは Repository、読み込みは QueryService | accepted |
| [0026](0026-system-query-dml-category.ja.md) | CQRS 分割外の第 4 DML カテゴリとして system_cqrs を導入する | accepted |
| [0027](0027-transaction-retry-idempotent-callers.ja.md) | シリアライゼーション競合時にトランザクションをリトライする; 呼び出し元のべき等性を要求する | accepted |
| [0028](0028-in-database-full-text-search.ja.md) | GIN trgm インデックス付き GENERATED STORED 列を使用してデータベースで全文検索を実行する | accepted |
| [0029](0029-uuidv7-identifiers.ja.md) | すべてのエンティティ主キーに UUIDv7（時刻順）識別子を使用する | accepted |
| [0030](0030-uber-fx-di.ja.md) | 依存性注入とライフサイクル管理に Uber Fx を採用する | accepted |
| [0031](0031-fx-neutral-di-abstraction.ja.md) | 中立的な DI 抽象（Registrar / Shutdowner）で fx を閉じ込める | accepted |
| [0032](0032-env-gated-wiring.ja.md) | DI で環境ごとに実装を切り替える（env-gated wiring） | accepted |
| [0033](0033-subsystem-typed-config-loaders.ja.md) | サブシステムスコープの envPrefix 型付き設定ローダー | accepted |
| [0034](0034-config-default-vs-required-governance.ja.md) | ガバナンス: デフォルト値はコード内（不変）vs 必須値はファイル内（可変） | accepted |
| [0035](0035-immutable-fail-fast-config.ja.md) | 設定は不変、起動時に一度だけ読み込み、フェイルファスト | accepted |
| [0036](0036-embedded-self-contained-binary.ja.md) | go:embed で設定（.env）とマイグレーションをバンドルし、自己完結型バイナリを生成する | accepted |
| [0037](0037-apperror-protocol-agnostic-errors.ja.md) | プロトコル非依存の集約エラー分類（apperror） | accepted |
| [0038](0038-broker-agnostic-worker-scaffold.ja.md) | ブローカー非依存のプルアック型ワーカースキャフォールド | accepted |
| [0039](0039-out-of-scope-push-streaming-brokers.ja.md) | プッシュ型ブローカーとストリーミングログプラットフォームはワーカーポートのスコープ外 | accepted (exclusion) |
| [0040](0040-sqs-adapter-opt-in.ja.md) | SQS アダプターはオプトインであり、デフォルトバイナリにはリンクされない | accepted |
| [0041](0041-transactional-outbox.ja.md) | トランザクショナルアウトボックス: ビジネストランザクション内でイベントを発行する | accepted |
| [0042](0042-at-least-once-outbox-poll.ja.md) | ポーリングによる少なくとも 1 回の配信（トランスポートレベルのリトライは無効） | accepted |
| [0043](0043-skip-locked-outbox-relay.ja.md) | SELECT FOR UPDATE SKIP LOCKED を使用した単一トランザクションリレー（複数インスタンス間で安全） | accepted |
| [0044](0044-message-id-idempotency-propagation.ja.md) | アウトボックスの message_id を受信者の Idempotency-Key として伝搬する | accepted |
| [0045](0045-outbox-dead-after-max-attempts.ja.md) | MaxAttempts = 10、以降メッセージはデッド状態（手動リプレイまで終端） | accepted |
| [0046](0046-outbox-retention-gc.ja.md) | 公開済み行の 7 日間保持 GC（10,000 件バッチ） | accepted |
| [0047](0047-publisher-http-profile-isolation.ja.md) | パブリッシャーの非標準 HTTP プロファイルをリレー内部に分離する | accepted |
| [0048](0048-relay-resident-gc-oneshot.ja.md) | リレーは常駐プロセス; GC はワンショット cron ジョブ | accepted |
| [0049](0049-single-tx-at-most-once-idempotency.ja.md) | 最大 1 回のセマンティクスのためにクレーム・ビジネス関数・完了を単一トランザクションで実行する | accepted |
| [0050](0050-idempotency-scope-required.ja.md) | ユーザー間のキー衝突を防ぐためにすべての Store 呼び出しに明示的なスコープを要求する | accepted |
| [0051](0051-idempotency-fixed-ttl.ja.md) | べき等キーの TTL を 24 時間に固定し、ルートごとの設定を持たない | accepted |
| [0052](0052-idempotency-response-persistence.ja.md) | 決定論的リプレイを可能にするためにレスポンスボディを JSON として永続化する（PII トレードオフを受容） | accepted |
| [0053](0053-idempotency-gc-separate-job.ja.md) | べき等キーのガベージコレクションを別途ワンショット CLI ジョブとして実行する | accepted |
| [0054](0054-idempotency-orthogonal-concerns.ja.md) | べき等性を楽観的ロックとレートリミットに対して直交性を保つ | accepted (exclusion) |
| [0055](0055-job-fresh-fx-app-per-run.ja.md) | 各ジョブ起動時に新鮮な fx.App を構築する（ワンショットライフサイクル） | accepted |
| [0056](0056-job-no-worker-machinery.ja.md) | ジョブは意図的にブローカー、サーキットブレーカー、ドレイン、ヘルスチェック機構を持たない | accepted (exclusion) |
| [0057](0057-job-explicit-registration.ja.md) | ジョブは明示的に登録される（自動探索なし） | accepted |
| [0058](0058-config-driven-observability-gating.ja.md) | 設定駆動の可観測性ゲーティング | accepted |
| [0059](0059-vendor-neutral-otlp-export.ja.md) | ベンダーニュートラルな OTLP のみのエクスポート（バックエンドは Collector に委譲） | accepted |
| [0060](0060-official-otel-semconv.ja.md) | 公式 OpenTelemetry セマンティック規約のみを使用する; カスタム semconv を発明したりベンダーキーを型付き設定に入れたりしない | accepted (exclusion) |
| [0061](0061-dual-path-metrics.ja.md) | メトリクスは 2 つのパスを経由 — OTLP プッシュと Prometheus スクレイプ | accepted |
| [0062](0062-lifecycle-independent-provider.ja.md) | 可観測性プロバイダーはライフサイクル非依存（ProviderShutdowner） | accepted |
| [0063](0063-fixed-default-sampling.ja.md) | SDK デフォルトサンプリングを固定する; サンプリングを環境変数として公開しない | accepted (exclusion) |
| [0064](0064-library-selection-policy.ja.md) | 単一責任ライブラリ選択ポリシー | accepted |
| [0065](0065-bridge-instrumentation-exceptions.ja.md) | ブリッジ / インストゥルメンテーションライブラリを有界な SRP 例外とする | accepted |
| [0066](0066-containerized-pinned-toolchain.ja.md) | 再現性のために mise でピン留めされたコンテナ化ツールチェーンを使用する | accepted |
| [0067](0067-mise-ssot-drift-gate.ja.md) | mise.toml は唯一の信頼できる情報源; バージョンは CI ドリフトゲートとともに下流に伝播する | accepted |
| [0068](0068-make-single-entrypoint.ja.md) | Make は .mk 登録と自己文書化ヘルプを持つ単一ツールエントリーポイント | accepted |
| [0069](0069-scripts-in-node-go.ja.md) | 運用スクリプトは scripts/ に Node（.mjs）または Go として配置する; シェルスクリプトは使用しない | accepted |
| [0070](0070-docker-compose-dev-environment.ja.md) | ローカル開発環境はプロファイル分離サービスを持つ Docker Compose で提供される | accepted |
| [0071](0071-two-layer-golangci-config.ja.md) | 2 層 golangci 設定: 最小限のデフォルト vs 完全な権威ゲート | accepted |
| [0072](0072-local-hooks-mirror-ci.ja.md) | ローカル git フックが CI コントラクトを複製する（ローカル == CI、グロブスコープ、バイパス後に一度検証） | accepted |
| [0073](0073-coverage-hard-gate.ja.md) | 合計カバレッジ 90% は CI ハードゲート、例外ガバナンスパスあり | accepted |
| [0074](0074-ci-real-graph-boot-check.ja.md) | CI は実際の Postgres に対して実際の fx グラフを起動する（起動検証） | accepted |
| [0075](0075-generated-artifact-drift-gate.ja.md) | 生成アーティファクトドリフトゲート + リリースブランチ集中型自動生成ボット | accepted |
| [0076](0076-multi-layer-security-scanning.ja.md) | 多層セキュリティスキャニング（到達可能性フィルタ済み govulncheck + スケジュール済み CodeQL SAST + シークレット + fs スキャン） | accepted |
| [0077](0077-sha-pinned-actions.ja.md) | サプライチェーン隔離で GitHub Actions を SHA でピン留めする | accepted |
| [0078](0078-rollback-integration-tests.ja.md) | センチネルエラーロールバックで実際の DB に対してインフラ統合テストを実行する | accepted |
| [0079](0079-multi-model-adversarial-review.ja.md) | ファインダーとバリファイアーサブエージェントを使用したマルチモデル敵対的レビューを使用する | accepted |
| [0080](0080-lean-a-spec-scaffold.ja.md) | スペックファイルからドメインとユースケースのみをスキャフォールドする; コントローラーとインフラは生成コードから導出する | accepted |
| [0081](0081-cli-humble-object-split.ja.md) | CLI humble-object 分割（薄い cmd/ シェル + テスト可能な internal/cli コア） | accepted |
| [0082](0082-single-multi-command-binary.ja.md) | すべてのロールは 1 つのマルチコマンドバイナリ | accepted |
| [0083](0083-single-runtime-image.ja.md) | コマンドオーバーライドを持つ単一ランタイムイメージ（用途別イメージなし） | accepted |
| [0084](0084-hardened-alpine-runtime.ja.md) | 強化 alpine ランタイムベースを使用する; distroless/scratch は使用しない | accepted (exclusion) |
| [0085](0085-per-environment-images.ja.md) | 環境別イメージ（.env マトリクス x APP_ENV ビルド引数、ビルド時に固定） | accepted |
| [0086](0086-predeploy-oneshot-migration.ja.md) | マイグレーションはデプロイ前ワンショットとして実行する; アプリケーション起動時に自動マイグレーションしない | accepted (exclusion) |
| [0087](0087-release-image-supply-chain.ja.md) | リリースイメージサプライチェーン整合性（cosign 署名 + プロベナンス + SBOM） | accepted |
| [0088](0088-vendor-neutral-deploy-skeleton.ja.md) | デプロイはベンダーニュートラルなスケルトン（ビルド/署名は実装済み; クラウド CD はテンプレート; レジストリは固定なし） | accepted |
| [0089](0089-docs-via-github-pages.ja.md) | 静的 docs/ を GitHub Pages で公開する（本番プッシュ時にリリース） | accepted |
| [0090](0090-no-in-app-rate-limiter.ja.md) | アプリケーション内レートリミッターを提供しない | accepted (exclusion) |
| [0091](0091-scheduled-job-concurrency-delegated.ja.md) | スケジュールジョブの並行処理をアプリ内で制御しない; スケジューラーに委譲する | accepted (exclusion) |
| [0092](0092-no-generic-cache-abstraction.ja.md) | 汎用 Cache 抽象を提供しない | accepted (exclusion) |

フロントマターフィールド: `status`、`date`、`deciders`、`supersedes` / `superseded-by`、`tags`。
Consequences は MADR 標準に従う（`Positive` / `Negative`; 任意で `Neutral`）。
