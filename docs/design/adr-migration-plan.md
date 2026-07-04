# ADR 移行 作業計画書（細粒度版）

> 一時文書（provisional）。`docs/decisions.md` を per-file ADR 群へ移行するための
> 作業計画。移行完了後に削除する。**このファイル単体で（会話コンテキストが失われても）
> 各 ADR 作成を再開できる**よう、決定ごとに「作成時に再読すべきファイル」を記載する。

## 背景と目的

`docs/decisions.md` は約 8 決定＋依存インベントリを 1 ファイルに集約しているが、
(1) その場編集で決定履歴が失われ（例: observability セクションは `OBSERVABILITY_ENABLED`
撤去時に上書きされ旧設計の理由が消えた）、(2) 不変であるべき決定と `go.mod` 追従の
依存表が混在して依存表が陳腐化した（`net/http/otelhttp` / `otel/sdk/log` 欠落）。

本リポジトリはテンプレートであり、利用者は fork して各決定を個別に supersede したい。
per-file ADR にすれば、モノリスを触らず 1 ファイル追加で個別 supersede できる。

## 移行方針

- **形式**: MADR-lite（frontmatter に `status` / `date` / `supersedes` /
  `superseded-by` / `tags`、本文は Context / Decision / Consequences / Alternatives）。
- **配置（最終）**: `docs/adr/NNNN-kebab-title.md`（4 桁採番、番号は再利用しない）。
  ja ミラーは `docs/ja/adr/`。
- **不変運用**: `accepted` 後は本文を編集せず、supersede は新 ADR 追加＋旧 ADR の
  status 変更で行う。
- **分類**（移す前に種別を確定する）:
  - **decision / exclusion** → ADR（`docs/adr/`）。
  - **rule**（日々強制される帰結）→ `docs/rules.md` に残し ADR へ backlink。
  - **inventory**（コード追従の目録）→ `docs/reference/dependencies.md`。ADR にしない。

## 粒度・採番・順序方針

- **細粒度＝決定単位。約 92 本。** サブシステム（outbox / idempotency / job /
  observability）はサブ決定ごとに独立 ADR へ分割する。
- **順序は依存・基礎度順**に採番する：基礎原則 → 契約(OpenAPI) → HTTP 層 → 永続化 →
  DI/config/errors → 非同期サブシステム → observability → ライブラリ/ビルド/CI →
  バイナリ/イメージ/デプロイ → プロジェクト全体の除外。番号は暫定で Phase 0 で確定
  （発見順ではなく本順序を正とする）。
- 種別: `migrate`＝decisions.md から移設 / `new`＝未文書化の潜在決定 /
  `exclusion`＝負の決定。★＝最優先（明確な代替案＋影響大＋現状ゼロ）。

## 各 ADR 作成時の共通参照（全 ADR で必ず再読）

どの ADR を書くときも先に以下を読むこと。

- `CLAUDE.md` — 改変スコープ・生成物・言語規則（可視出力は日本語）。
- `docs/rules.md` — 非交渉ルール。ADR は rule と重複させず backlink を張る。
- `docs/decisions.md` — 移設元の原文（migrate 種別はここが本文ソース）。
- `docs/adr/README.md` / `docs/adr/template.md` — Phase 0 で作る形式・採番・分類規約。
- 対象サブシステムの `docs/design/<name>.md` と該当パッケージ `README.md`。

## 移行対象 ADR 一覧（依存・基礎度順）

「作成時に再読」は共通参照に**加えて**読むファイル。

### 基礎原則

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0001 | ロックイン回避を設計原則にする(ベンダ/ライブラリ置換可能性) | new | architecture.md:94-104 / project/policy.md / decisions.md:226-236 |
| 0002 | 実用的オニオンアーキテクチャの採用 | migrate | decisions.md:23-61 / architecture.md / rules.md:9-36 |
| 0003 | 境界を interface で定義し疎結合化(DIP) | new | architecture.md(dependency inversion) / rules.md:9-36。※0002 と統合可 |
| 0004 | モジュラモノリス(マイクロサービス非目標) | new | architecture.md:253-266 / project/scope.md:46-57 |
| 0005 | REST/Worker/Job は駆動アダプタ(分割軸にしない) | new | design/rest.md:11 / design/README.md:16-18 |
| 0006 | 構造安全をツール＋CI(depguard)で強制 | new | architecture.md:77-92 / .golangci.yml / rules.md:49-77 |
| 0007 | with-AI 開発方式(AGENTS.md を運用契約に) | new | AGENTS.md / CLAUDE.md / .claude/ |
| 0008 | docs を正典とする戦略(EN 正典＋ja mirror＋portal) | new | docs/index.md / docs/maintenance/docs-structure.md / docs/portal/manifest.yaml |

### 契約・OpenAPI

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0009 | OpenAPI-first API 契約 | migrate | decisions.md:63-90 / rules.md:78-93 |
| 0010 | Redocly モジュール分割＋bundle→生成 のスペック工程 | new | openapi/README.md:13-33,58-74 / redocly.yaml / .makefiles/openapi/gen.mk |
| 0011 | tag/handler 単位 oapi-codegen 生成＋strict-server モード | new | handler/**/*_handler.go:1-2(//go:generate) / handler/**/gen/server.gen.go |
| 0012 | spec 駆動リクエスト検証＋認証(security:=強制)／レスポンス実行時検証なし | new | httpstack/oapi/oapi.go:17-39 / openapi/README.md:122 / openapi/boundary-ownership.md:30 |
| 0013 | 境界値オーナーシップ(OpenAPI=ワイヤ契約≠ドメイン規則、request⊆domain⊆response) | new | openapi/boundary-ownership.md:7-8,21-27,50-56 |
| 0014 | `/metrics` の認証例外(OpenAPI 検証外・別 Echo BasicAuth) | new | openapi/README.md:122 / httpstack。※0012 に統合可 |

### HTTP 層

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0015 | Echo を HTTP フレームワークに採用 | migrate | decisions.md:146-170 |
| 0016 | 優先度順(データ駆動)ミドルウェア連鎖 | new | design/rest.md:29,244 / controller/httpstack/README.md |
| 0017 | 外向き HTTP レジリエンス基盤(retry/CB/budget/dual timeout) | new ★ | infrastructure/httpclient/README.md:5 |
| 0018 | egress SSRF/dial ガードのセキュリティ姿勢 | new ★ | observability/http_client_transport.go:18-23 / httpclient/README.md / outbox/http_publisher.go:30 |

### 永続化・データ

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0019 | SQL-first データアクセス | migrate | decisions.md:92-117 |
| 0020 | sqlc による型安全 SQL 生成 | migrate | decisions.md:119-144 |
| 0021 | merge-dml＋dump-schema →『database/gen/』(schema.gen.sql) を sqlc 単一入力に | new | .makefiles/database/dml-merge.mk:47-48 / gen.mk:27-34 / sqlc.yaml:4-5 / database/README.md:26-47 |
| 0022 | append-only な不変マイグレーション | new | rules.md:94-111 / development-flow.md |
| 0023 | migration ID は連番(6桁)＋gap/pair を CI 強制(timestamp 不採用) | new | database/migrations/README.md:53-69 / migration-check.yaml。0022 は不変性、本 ADR は採番規律 |
| 0024 | マスタデータは migration・トランザクション seed は seed/(本番除外) | new | database/seed/README.md:47-58 / database/README.md / migrations 000003,000005,000008 |
| 0025 | 軽量 CQRS(Repository=書込 / QueryService=読取 / command_service も配線) | migrate/new ★ | infrastructure/rdb/query_service/README.md:40-47 / di/module/persistence.go:36 / rules.md:158-169 |
| 0026 | system_query を非CQRS 第4 DML カテゴリに(health/idempotency/outbox) | new | database/dml/README.md:23-31 / database/dml/system_query/README.md |
| 0027 | 競合時の tx 再試行＋呼び出し側冪等契約 | new ★ | usecase/boundary/tx/README.md:16-18 / infrastructure/rdb/driver/transaction.go:79,116 |
| 0028 | 全文検索は DB 内(GENERATED STORED 列＋GIN pg_trgm、query_service 経由) | new | migrations/000011_users_table_search_text_column.up.sql / dml/query_service/user/select_users_by_keyword.sql |
| 0029 | UUIDv7(時間順)識別子戦略 | new | pkg/uuid/README.md:5,11 / sqlc.yaml(型 override) |

### DI・config・errors

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0030 | Uber Fx を DI＋ライフサイクルに採用 | migrate | decisions.md:172-197 |
| 0031 | fx を中立 DI 抽象で封じ込め(Registrar/Shutdowner) | new | di/lifecycle/README.md / di/shutdowner/README.md |
| 0032 | DI で環境ごとに実装を差し替える(env-gated wiring) | new | di/module/authz.go:29-47 / di/module/README.md / config の Env 定数 |
| 0033 | サブシステム別 envPrefix 型付きローダ | new | internal/config/envspec.go:5-19 / env/README.md:9-13 |
| 0034 | default-in-code(不変) vs required-in-file(可変) ガバナンス | new | env/README.md:16,22-24 / internal/config/envspec.go |
| 0035 | 不変・起動時一括・fail-fast な型付き config | new | config/README.md / config/config.go |
| 0036 | go:embed で config(.env)/migration を同梱＝自己完結バイナリ | new | embed.go:7 / config/README.md |
| 0037 | プロトコル非依存の集約エラー分類(apperror) | new | apperror/README.md:5,8 / rules.md:215-221 |

### 非同期サブシステム

worker:

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0038 | broker 非依存 pull-ack worker scaffold | migrate | decisions.md:199-224 / design/worker.md |
| 0039 | push/streaming broker を対象外 | exclusion | decisions.md:209,222 / design/worker.md |
| 0040 | SQS アダプタは opt-in / 既定バイナリ非リンク | migrate | decisions.md:212,223-224 / infrastructure/queue/sqs/README.md |

outbox（design/outbox.md 全体が decisions.md に欠落。:5 のリンク切れも解消）:

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0041 | トランザクショナル outbox(業務 tx 内で emit) | new | design/outbox.md:11,35 |
| 0042 | at-least-once poll(transport retry 無効 / D10) | new | design/outbox.md:36,263 |
| 0043 | SKIP LOCKED 単一 tx relay(多インスタンス安全) | new | design/outbox.md:37 / database/dml/system_query/outbox/claim_pending_outbox.sql |
| 0044 | message_id を受信側 Idempotency-Key に伝播 | new | design/outbox.md:38 |
| 0045 | MaxAttempts=10 で dead(人手 replay まで終端) | new | design/outbox.md:52,256 |
| 0046 | 7d retention GC(batch 10,000) | new | design/outbox.md:54,259 / .../outbox/delete_published_outbox.sql |
| 0047 | publisher の非標準 HTTP プロファイルを relay に隔離 | new | design/outbox.md:176,268 / di/outboxrelay |
| 0048 | relay は常駐 / GC は one-shot cron | new | design/outbox.md:27 |

idempotency:

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0049 | claim/businessFn/complete を単一 tx で at-most-once | new | design/idempotency.md:28,13 |
| 0050 | 全 Store 呼び出しで scope 必須(IDOR 防止) | new | design/idempotency.md:29 |
| 0051 | TTL 24h 固定(per-route 設定なし) | new | design/idempotency.md:227-229,252 |
| 0052 | レスポンス本体を JSON 保存(PII トレードオフ) | new | design/idempotency.md:231 |
| 0053 | idempotency GC を別 one-shot ジョブ化 | new | design/idempotency.md:23,230 |
| 0054 | 楽観ロック/レート制限と直交(opt-in) | exclusion | design/idempotency.md:13 |

job:

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0055 | 起動毎に fx.App を新規構築(one-shot) | new | design/job.md:24 |
| 0056 | broker/circuit/drain/health を持たない(worker と対照) | exclusion | design/job.md:24 |
| 0057 | ジョブは明示登録(auto-discovery なし) | new | design/job.md:178 |

### observability

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0058 | config 駆動 observability gating | migrate | decisions.md:272-294 / design/observability.md:16 |
| 0059 | ベンダ中立 OTLP-only エクスポート(Collector 委譲) | new | design/observability.md:15 / observability/README.md |
| 0060 | 公式 OTel semconv を使用(custom なし・vendor キーを typed config に入れない) | exclusion | internal/observability/provider.go:23,58-60 / observability/README.md:46-51 |
| 0061 | メトリクス 2 経路(OTLP push＋Prometheus scrape) | new | design/observability.md:146-153 |
| 0062 | provider をライフサイクル非依存に(ProviderShutdowner) | new | design/observability.md:30 |
| 0063 | SDK 既定サンプリング固定(env 非設定) | exclusion | design/observability.md:80 |

### ライブラリ・ビルド・ツールチェーン

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0064 | 単一責務ライブラリ選定ポリシー | migrate | decisions.md:226-236 |
| 0065 | bridge/instrumentation 例外(SRP 例外) | migrate | decisions.md:296-314 |
| 0066 | コンテナ化ツールチェーン＋mise 固定(再現性) | new | rules.md:281-296 / makefile / .makefiles/ / mise.toml |
| 0067 | mise SSOT のダウンストリーム伝播＋CI ドリフトgate | new | mise.toml:1-7,47-49 / sync-versions-check.yaml:39-61 / scripts/sync-versions |
| 0068 | Make を単一ツールエントリポイントに(.mk 登録＋自己文書化 help 契約) | new | makefile:6-64 / scripts/make_help.mjs:23-53 |
| 0069 | 運用スクリプトは scripts/ に Node(.mjs)/Go で置き sh 不採用 | new | scripts/README.md / scripts/。script↔pkg↔internal の役割分離も明記 |
| 0070 | ローカル開発環境を docker-compose で提供(tool-runner＋ビューア群) | new | docker-compose.yaml / docker/ |

### CI・品質ゲート

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0071 | 2層 golangci 設定(minimal 既定 vs full 権威ゲート) | new | .golangci.yaml / .golangci-full.yaml / .makefiles/go/golangci-lint.mk:8,11 |
| 0072 | ローカル git hook が CI 契約を複製(local==CI・glob 限定・bypass-then-verify-once) | new | .lefthook.yaml:1-55 |
| 0073 | 総カバレッジ90% を CI ハードゲート化＋例外ガバナンス | new | .makefiles/go/test.mk:11-12,44-51 / internal/observability/README.md:504-523 / go-test.yaml:79-80 |
| 0074 | CI で実 fx グラフ＋実 Postgres を起動検証 | new | app-di-startup-check.yaml:58-83 / worker-boot-check.yaml:60-82 / job-boot-check.yaml:59-81 |
| 0075 | 生成物ドリフトゲート＋リリースブランチ集中自動生成 bot | new | gen-go-artifacts-check.yaml:36-111 / gen-db-artifacts-check.yaml / auto-generate-docs.yaml:90-181 |
| 0076 | 多層セキュリティスキャン(到達可能性フィルタ govulncheck＋定期 CodeQL SAST) | new | code-ql.yaml:21-22 / vulnerability-check.yaml:57-61 / secret-scan.yaml / trivy-fs.yaml |
| 0077 | GitHub Actions を SHA ピン＋供給網検疫で固定 | new | .github/actions-pin.toml / scripts/pin-actions / .github/workflows |

### 開発プロセス・品質

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0078 | 実DB always-rollback 統合テスト(sentinel error でロールバック、mock しない) | new | internal/infrastructure/rdb/testkit/README.md:1-257 / test_kit.go |
| 0079 | 多モデル敵対レビュー(reviewer≠implementer、finder→verifier＋runtime-gap) | new | .claude/agents/adversarial-reviewer.md:1-8 / review-verifier.md / skills/local-review/SKILL.md |
| 0080 | spec 駆動 lean A scaffold(domain・usecase のみ spec、controller・infra は導出) | new | .claude/scaffold-spec/*.md / scaffold-endpoint / scaffold-controller / scaffold-infra-db |

### バイナリ・イメージ・デプロイ

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0081 | CLI humble-object 分割(thin cmd/ + testable internal/cli 中核) | new | internal/cli/README.md:49-82 / cmd/commands.go:6-19 / cmd/outbox_relay.go:28-43 |
| 0082 | 全ロールを単一マルチコマンドバイナリに集約 | new | cmd/main.go:13-31 / cmd/commands.go:7-18 |
| 0083 | 単一ランタイムイメージ＋コマンド上書き(用途別イメージを作らない) | new | docker/server/Dockerfile:55-57 / docker/server/README.md:24-26 |
| 0084 | hardened-alpine ランタイム基盤(distroless/scratch 不採用) | exclusion | docker/server/Dockerfile:42-53 / docker/server/README.md:16,21 |
| 0085 | per-env イメージ(.env マトリクス×APP_ENV build-arg、ビルド時固定) | new | docker/server/Dockerfile:25-30 / deploy-app.yaml:54-63 / env/README.md:176 |
| 0086 | マイグレーションは pre-deploy one-shot(起動時 auto-migrate 禁止) | exclusion | deploy-app.yaml:192-204 |
| 0087 | リリースイメージの供給網完全性(cosign 署名＋provenance＋SBOM) | new | deploy-app.yaml:131-168 |
| 0088 | デプロイはベンダ中立スケルトン(build/sign 実装・cloud CD 雛形・registry 非固定) | new | deploy-app.yaml:67-72,96-102,181-218 |
| 0089 | 静的 docs/ を GitHub Pages で公開(production push で発行) | new | deploy-docs.yaml:1-45。0008 と対 |

### プロジェクト全体の除外（負の ADR）

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0090 | アプリ内レート制限器を持たない | exclusion | project/out-of-scope.md:11-16 |
| 0091 | scheduled job 並走制御はスケジューラ委譲 | exclusion | project/out-of-scope.md:17-26 |
| 0092 | 汎用 Cache 抽象を持たない | exclusion | project/out-of-scope.md:49-57 |

### 除外(負の)ADR とリポジトリセットアップ

exclusion 種別（0039 push/streaming / 0054 idempotency 直交 / 0056 job 機構除外 /
0060 公式 semconv / 0063 サンプリング固定 / 0084 hardened-alpine / 0086 pre-deploy
migration / 0090 rate-limiter / 0091 scheduled-job / 0092 Cache）は、テンプレの
**意図的な非選択**であり、fork 利用者が最も方針転換したい箇所。

- 各 exclusion ADR に `tags: [exclusion, setup-review]` を付与し、セットアップ導線から
  機械的に列挙できるようにする。
- リポジトリセットアップ（`make setup-repo` 系 / `docs/get-started/setup-repository.md`）に
  「exclusion ADR をレビューし、**受容**するか **supersede**（自分の決定を新 ADR として
  追加し旧を `superseded` にする）か決める」ステップを追加する。実装は Phase 4（exclusion
  ADR 作成）以降に setup フロー拡張として別途。
- 元 ADR は消さず supersede する（ADR 不変運用と整合）。この「exclusion ADR を setup 時の
  上書きポイントにする」方針自体も 1 本の ADR に昇格させる余地あり（要判断）。

### ADR にしないもの

- **rule（rules.md に残す＋backlink）**: 層依存方向 / usecase-境界依存 / 生成コード
  非編集 / ドメイン純粋性 / context 伝播 / DTO 境界変換 / tx 配置 / エラー処理機構 /
  comment・doc・testing 規約 / 薄いハンドラ / pkg 相互独立 / 実効的不変性 /
  idempotency claim 機構 / RED メトリクス基数制限 / OpenAPI casing・URL versioning 規約 /
  secret 階層 / mock 生成方針。
- **inventory（生きた文書へ）**: 直接依存表 → `docs/reference/dependencies.md`
  （`net/http/otelhttp` / `otel/sdk/log` の欠落もここで修正）/ ミドルウェア優先度
  定数表 → REST 設計文書に残す / sqlc 生成設定 / ワークフロー標準エンベロープ。
- **棄却した除外候補（out-of-scope に列挙のみ／代替案が薄く ADR 化しない）**:
  デプロイ実装 / IaC / o11y 運用設定 / circuit breaker / secret rotation / 監査ログ /
  RBAC / セッション管理 / PII 暗号化 / 認証機構 / アカウントロックアウト /
  データエクスポート・削除。

## フェーズ別作業手順（種別ベース）

番号ではなく種別で駆動するため、上表の並べ替え・採番変更に影響されない。

### Phase 0: 型固定

`docs/adr/` に README（決定ログ＋分類規約＋supersede 運用）/ template.md（MADR-lite）/
`0000-record-architecture-decisions.md`（メタ ADR）/ 見本 1 本（0001 ロックイン回避）を作成し、
形式・採番・supersede 運用を確定する。ここで本計画の依存・基礎度順を正式採番に落とす。

### Phase 1: migrate 種別を全移設

decisions.md 記載の既存決定（onion / OpenAPI-first / SQL-first / sqlc / Echo / Fx /
library-selection-policy / bridge 例外 / worker-scaffold / SQS-opt-in / o11y-gating の
11 本）を 1:1 で ADR 化。内容は移すだけ。

### Phase 2: ★（強い新規）

軽量 CQRS(0025) / tx 再試行契約(0027) / HTTP レジリエンス(0017) / SSRF ガード(0018)。
現状ゼロで影響大のものを優先。

### Phase 3: 残りの new を依存順に作成

基礎原則 → 契約 → HTTP → 永続化 → DI/config → 非同期サブシステム → observability →
ライブラリ/ビルド/CI → 開発プロセス → バイナリ/デプロイ の順で new 種別を作成。
outbox 作成時に `design/outbox.md:5` のリンク切れを解消。

### Phase 4: exclusion（負の ADR）

push/streaming 除外 / idempotency 直交 / job 機構除外 / サンプリング固定 /
公式 semconv / hardened-alpine / pre-deploy migration / rate-limiter / Cache /
scheduled-job 並走。out-of-scope.md 等は目録として残す。

### Phase 5: インベントリ分離・参照貼替・撤去

依存表を `docs/reference/dependencies.md` へ移し欠落修正。全参照貼替、ja 同期、
`docs/decisions.md` 撤去（または 1 行リダイレクト）。

## 参照貼り替え対象（Phase 5）

`docs/decisions.md` は約 20 ファイルから参照される。

- canonical: README(en/ja) / rules.md / index.md / maintenance/docs-structure.md
- design: design/README.md / worker.md / outbox.md（および ja）
- portal: portal/guides/(overview / worker-design / outbox-design, en/ja)
- ツール: `.claude/agents/doc-reviewer.md`
- **AGENTS.md**: 人間専用。参照更新は**人手**で行う。

## 再開手順（コンテキストなしで着手する場合）

1. 本ファイルと「各 ADR 作成時の共通参照」を読む。
2. Phase 0 が未了なら先に型固定を行う。
3. 作る ADR を上表から選び、「作成時に再読」＋共通参照を読んでから template で起票。
4. migrate は decisions.md 原文を Context/Decision/Consequences/Alternatives へ再配置。
   new/exclusion は出所 design doc / README / コードから同構造で起こす。
5. rule と重複させない。関連 rule には backlink を張る。
6. 進捗は本表の状態でなく PR の追加コミットで追う（PR は base `release/v2.0.1`）。

## リスクと注意点

- **AGENTS.md は AI 編集禁止。** Phase 5 の参照更新は人手前提。
- **ja 二重管理。** 各 ADR は `docs/ja/adr/` にミラー要（`canonicalize-doc`）。
- **markdownlint。** 見出しに `<...>` 形式は HTML タグ扱いで MD024 誤検知、コード
  フェンスは言語必須。ローカル mermaid lint は環境依存で失敗（内容問題ではない）。
- **粒度。** 本版は細粒度（約 92 本）。粗くしたい場合はサブシステム単位に束ね直す。
- **統合候補。** 0003(DIP)↔0002(onion) / 0014(/metrics 例外)↔0012(検証) は統合可。

## 完了条件

- 約 92 本の ADR が `docs/adr/` に存在し、種別分類が反映されている。
- 依存表が `docs/reference/dependencies.md` に分離され欠落が修正されている。
- rules.md の該当ルールから対応 ADR へ backlink がある。
- `docs/decisions.md` への全参照が貼替済み（AGENTS.md は人手更新）。
- ja ミラー同期、`docs/decisions.md` が撤去またはリダイレクト化されている。
