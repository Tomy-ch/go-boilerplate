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

## 粒度方針（本版で確定）

**細粒度＝決定単位。約 59 本。** サブシステム（outbox / idempotency / job / observability）は
サブ決定ごとに独立 ADR へ分割する（retention・MaxAttempts・TTL 等も各 1 本）。fork 側が
サブ決定を個別に supersede できることを優先する。

## 各 ADR 作成時の共通参照（全 ADR で必ず再読）

どの ADR を書くときも先に以下を読むこと。

- `CLAUDE.md` — 改変スコープ・生成物・言語規則（可視出力は日本語）。
- `docs/rules.md` — 非交渉ルール。ADR は rule と重複させず backlink を張る。
- `docs/decisions.md` — 移設元の原文（migrate 種別はここが本文ソース）。
- `docs/adr/README.md` / `docs/adr/template.md` — Phase 0 で作る形式・採番・分類規約。
- 対象サブシステムの `docs/design/<name>.md` と該当パッケージ `README.md`。

## 移行対象 ADR 一覧・採番・参照

種別: `migrate`＝decisions.md から移設 / `new`＝未文書化の潜在決定 /
`exclusion`＝負の決定。★＝最優先（明確な代替案＋影響大＋現状ゼロ）。
「作成時に再読」は共通参照に**加えて**読むファイル。

### 基盤・アーキテクチャ

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0001 | 実用的オニオンアーキテクチャの採用 | migrate | decisions.md:23-61 / architecture.md / rules.md(層依存 9-36) |
| 0002 | モジュラモノリス（マイクロサービス非目標） | new | architecture.md:253-266,294-303 / project/scope.md:35,46-57 |
| 0003 | 構造安全をツール＋CI(depguard)で強制 | new | architecture.md:77-92 / .golangci.yml(depguard) / rules.md:49-77 |
| 0004 | REST/Worker/Job は駆動アダプタ（分割軸にしない） | new | design/rest.md:11 / design/README.md:16-18 / design/worker.md |
| 0005 | OpenAPI-first API 契約 | migrate | decisions.md:63-90 / rules.md:78-93 |

### 永続化

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0006 | SQL-first データアクセス | migrate | decisions.md:92-117 |
| 0007 | sqlc による型安全 SQL 生成 | migrate | decisions.md:119-144 |
| 0008 | append-only な不変マイグレーション | new | rules.md:94-111 / development-flow.md |
| 0009 | 軽量 CQRS（Repository=書込 / QueryService=読取 / CommandService も配線） | new ★ | infrastructure/rdb/query_service/README.md:40-47 / di/module/persistence.go:36 / rules.md:158-169 / usecase/README.md |
| 0010 | 競合時の tx 再試行＋呼び出し側冪等契約 | new ★ | usecase/boundary/tx/README.md:16-18 / infrastructure/rdb/driver/transaction.go:79,116 / design/outbox.md |
| 0011 | UUIDv7（時間順）識別子戦略 | new | pkg/uuid/README.md:5,11 / sqlc.yaml(型 override) |

### HTTP フレームワーク・基盤

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0012 | Echo を HTTP フレームワークに採用 | migrate | decisions.md:146-170 |
| 0013 | 優先度順（データ駆動）ミドルウェア連鎖 | new | design/rest.md:29,244 / controller/httpstack/README.md |
| 0014 | 外向き HTTP レジリエンス基盤(retry/CB/budget/dual timeout) | new ★ | infrastructure/httpclient/README.md:5 / httpclient 実装 |
| 0015 | egress SSRF / dial ガードのセキュリティ姿勢 | new ★ | observability/http_client_transport.go:18-23 / infrastructure/httpclient/README.md(security) / outbox/http_publisher.go:30 |

### DI・config・errors

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0016 | Uber Fx を DI＋ライフサイクルに採用 | migrate | decisions.md:172-197 |
| 0017 | fx を中立 DI 抽象で封じ込め(Registrar/Shutdowner) | new | di/lifecycle/README.md / di/shutdowner/README.md |
| 0018 | DI で環境ごとに実装を差し替える(env-gated wiring) | new | di/module/authz.go:29-47 / di/module/README.md / config の Env 定数(EnvLocal/EnvCI/EnvTest/本番相当) |
| 0019 | 不変・起動時一括・fail-fast な型付き config | new | config/README.md / config/config.go |
| 0020 | プロトコル非依存の集約エラー分類(apperror) | new | apperror/README.md:5,8 / rules.md:215-221 |

### ライブラリ・ツールチェーン

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0021 | 単一責務ライブラリ選定ポリシー | migrate | decisions.md:226-236 |
| 0022 | bridge/instrumentation 例外（SRP 例外） | migrate | decisions.md:296-314 |
| 0023 | コンテナ化ツールチェーン＋mise 固定(再現性) | new | rules.md:281-296 / makefile / .makefiles/ / mise.toml |

### worker

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0024 | broker 非依存 pull-ack worker scaffold | migrate | decisions.md:199-224 / design/worker.md |
| 0025 | push/streaming broker を対象外 | exclusion | decisions.md:209,222 / design/worker.md |
| 0026 | SQS アダプタは opt-in / 既定バイナリ非リンク | migrate | decisions.md:212,223-224 / infrastructure/queue/sqs/README.md |

### outbox（design/outbox.md 全体が decisions.md に欠落。:5 のリンク切れも解消）

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0027 | トランザクショナル outbox（業務 tx 内で emit） | new | design/outbox.md:11,35 |
| 0028 | at-least-once poll（transport retry 無効 / D10） | new | design/outbox.md:36,263 |
| 0029 | SKIP LOCKED 単一 tx relay（多インスタンス安全） | new | design/outbox.md:37 / database/dml/system_query/outbox/claim_pending_outbox.sql |
| 0030 | message_id を受信側 Idempotency-Key に伝播 | new | design/outbox.md:38 |
| 0031 | MaxAttempts=10 で dead（人手 replay まで終端） | new | design/outbox.md:52,256 |
| 0032 | 7d retention GC（batch 10,000） | new | design/outbox.md:54,259 / .../outbox/delete_published_outbox.sql |
| 0033 | publisher の非標準 HTTP プロファイルを relay に隔離 | new | design/outbox.md:176,268 / di/outboxrelay |
| 0034 | relay は常駐 / GC は one-shot cron | new | design/outbox.md:27 |

### idempotency

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0035 | claim/businessFn/complete を単一 tx で at-most-once | new | design/idempotency.md:28,13 |
| 0036 | 全 Store 呼び出しで scope 必須(IDOR 防止) | new | design/idempotency.md:29 |
| 0037 | TTL 24h 固定（per-route 設定なし） | new | design/idempotency.md:227-229,252 |
| 0038 | レスポンス本体を JSON 保存(PII トレードオフ) | new | design/idempotency.md:231 |
| 0039 | idempotency GC を別 one-shot ジョブ化 | new | design/idempotency.md:23,230 |
| 0040 | 楽観ロック/レート制限と直交（opt-in） | exclusion | design/idempotency.md:13 |

### job

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0041 | 起動毎に fx.App を新規構築(one-shot) | new | design/job.md:24 |
| 0042 | broker/circuit/drain/health を持たない(worker と対照) | exclusion | design/job.md:24 |
| 0043 | ジョブは明示登録（auto-discovery なし） | new | design/job.md:178 |

### observability

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0044 | config 駆動 observability gating | migrate | decisions.md:272-294 / design/observability.md:16 |
| 0045 | ベンダ中立 OTLP-only エクスポート(Collector 委譲) | new | design/observability.md:15 / observability/README.md |
| 0046 | メトリクス 2 経路(OTLP push＋Prometheus scrape) | new | design/observability.md:146-153 |
| 0047 | provider をライフサイクル非依存に(ProviderShutdowner) | new | design/observability.md:30 |
| 0048 | SDK 既定サンプリング固定(env 非設定) | exclusion | design/observability.md:80 |

### 除外（負の ADR）

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0049 | アプリ内レート制限器を持たない | exclusion | project/out-of-scope.md:11-16 |
| 0050 | scheduled job 並走制御はスケジューラ委譲 | exclusion | project/out-of-scope.md:17-26 |
| 0051 | 汎用 Cache 抽象を持たない | exclusion | project/out-of-scope.md:49-57 |

### 追加(2nd pass): 設計原則・運営・ツールチェーン・メタ

> 初回スキャンが `internal` / `pkg` / `docs` 中心で未カバーだった `.github` /
> `scripts` / `docker` / `.claude` / トップレベル思想からの追補。番号は暫定
> （Phase 0 で通し再採番）。

| # | ADR | 種別 | 作成時に再読 |
| --- | --- | --- | --- |
| 0052 | ロックイン回避を設計原則にする(ベンダ/ライブラリ置換可能性) | new | architecture.md:94-104 / project/policy.md / decisions.md:226-236。0021/0026/0045 の上位原則 |
| 0053 | 境界を interface で定義し疎結合化(DIP) | new | architecture.md(dependency inversion) / rules.md:9-36。※0001 onion と重複気味・統合可 |
| 0054 | with-AI 開発方式(AGENTS.md を運用契約に) | new | AGENTS.md / CLAUDE.md / .claude/ |
| 0055 | docs を正典とする戦略(EN 正典＋ja mirror＋portal、docs が agent を駆動) | new | docs/index.md / docs/maintenance/docs-structure.md / docs/portal/manifest.yaml |
| 0056 | GitHub Actions を SHA ピン＋供給網検疫で固定 | new | .github/actions-pin.toml / scripts/pin-actions / .github/workflows / (actions-pin skill) |
| 0057 | 運用スクリプトは scripts/ に Node(.mjs)/Go で置き sh 不採用 | new | scripts/README.md / scripts/(全 .mjs,.go) / rules.md(pkg↔internal 分離)。script↔pkg↔internal の役割分離も明記 |
| 0058 | go:embed で config(.env)/migration を同梱＝自己完結バイナリ | new | embed.go:7 / config/README.md。※0019 は不変性、本 ADR は同梱/自己完結の観点 |
| 0059 | ローカル開発環境を docker-compose で提供(tool-runner＋ビューア群) | new | docker-compose.yaml / docker/。※0023 はツール実行、本 ADR は dev stack 構成 |

### ADR にしないもの

- **rule（rules.md に残す＋backlink）**: 層依存方向 / usecase-境界依存 / 生成コード
  非編集 / ドメイン純粋性 / context 伝播 / DTO 境界変換 / tx 配置 / エラー処理機構 /
  comment・doc・testing 規約 / 薄いハンドラ / pkg 相互独立 / 実効的不変性 /
  idempotency claim 機構 / RED メトリクス基数制限。
- **inventory（生きた文書へ）**: 直接依存表 → `docs/reference/dependencies.md`
  （`net/http/otelhttp` / `otel/sdk/log` の欠落もここで修正）/ ミドルウェア優先度
  定数表 → REST 設計文書に残す。
- **棄却した除外候補（out-of-scope に列挙のみ／代替案が薄く ADR 化しない）**:
  デプロイ実装 / IaC / o11y 運用設定 / circuit breaker / secret rotation / 監査ログ /
  RBAC / セッション管理 / PII 暗号化 / 認証機構 / アカウントロックアウト /
  データエクスポート・削除。代替案分析を持つ 0049-0051 のみ ADR 昇格した。

## フェーズ別作業手順

### Phase 0: 型固定

`docs/adr/` に README（決定ログ＋分類規約＋supersede 運用）/ template.md（MADR-lite）/
`0000-record-architecture-decisions.md`（メタ ADR）/ 見本 1 本（0001 onion）を作成し、
形式・採番・supersede 運用を確定する。

### Phase 1: 既存決定の移設（migrate 種別）

0001,0005,0006,0007,0012,0016,0021,0022,0024,0026,0044 を decisions.md から 1:1 移設。

### Phase 2: 強い新規（★）

0009,0010,0014,0015 を作成。現状ゼロで影響大のものを優先。

### Phase 3: サブシステム・潜在決定

outbox(0027-0034) / idempotency(0035-0039) / job(0041,0043) / observability
追加(0045-0047)、および 0002-0004,0008,0011,0013,0017-0020,0023、さらに
0052-0059(設計原則・運営・ツールチェーン・メタ) を作成。outbox 作成時に
`design/outbox.md:5` のリンク切れを解消。

### Phase 4: 除外（負の ADR）

0025,0040,0042,0048,0049,0050,0051 を作成。out-of-scope.md 等は目録として残す。

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
- **粒度。** 本版は細粒度（約 59 本）。粗くしたい場合はサブシステム単位に束ね直す。

## 完了条件

- 約 59 本の ADR が `docs/adr/` に存在し、種別分類が反映されている。
- 依存表が `docs/reference/dependencies.md` に分離され欠落が修正されている。
- rules.md の該当ルールから対応 ADR へ backlink がある。
- `docs/decisions.md` への全参照が貼替済み（AGENTS.md は人手更新）。
- ja ミラー同期、`docs/decisions.md` が撤去またはリダイレクト化されている。
