# ADR 移行 作業計画書

> 一時文書（provisional）。`docs/decisions.md` を per-file ADR 群へ移行するための
> 作業計画。移行完了後に削除する。リポジトリ全体スキャン（canonical / design docs /
> exclusion / latent code の 4 面）で洗い出した候補にもとづく。

## 背景と目的

`docs/decisions.md` は約 8 決定＋依存インベントリを 1 ファイルに集約しているが、次の
2 点が問題化している。

- **その場編集で決定履歴が失われる。** 例：observability セクションは
  `OBSERVABILITY_ENABLED` 撤去時に上書きされ、旧設計を選んだ理由が消えた。
- **性質の異なる内容の混在。** 不変であるべき決定と、`go.mod` に追従して変化し続ける
  依存表が同居し、依存表が静かに陳腐化した（`net/http/otelhttp` / `otel/sdk/log`
  の欠落）。

本リポジトリはテンプレートであり、利用者は fork して各決定を個別に上書き
（supersede）したい。per-file ADR にすれば、モノリスを触らず 1 ファイル追加で
個別 supersede できる。

## 現状診断（要点）

- decisions.md 記載の決定は現状のコードと整合（onion / OpenAPI-first / sqlc / echo /
  fx / worker / o11y gating）。**思想面の陳腐化はない。**
- 陳腐化しているのは依存表のみ：直接依存 `net/http/otelhttp` `otel/sdk/log` が未記載
  （後述の通りインベントリは ADR 化せず別文書へ）。
- **outbox サブシステムが decisions.md に丸ごと欠落**。加えて
  `docs/design/outbox.md:5` が存在しない outbox ADR を指す**リンク切れ**がある。
- スキャンの結果、ADR 化に値する決定は **約 30 件**（既存移行 9＋新規 21）と判明。

## 移行方針

- **形式**：MADR-lite（frontmatter に `status` / `date` / `supersedes` /
  `superseded-by`、本文は Context / Decision / Consequences / Alternatives）。
- **配置（最終）**：`docs/adr/NNNN-kebab-title.md`（4 桁採番、番号は再利用しない）。
  今回の計画段階では実 ADR は作らず、本計画書のみ。
- **不変運用**：`accepted` 後は本文を編集せず、supersede は新 ADR 追加＋旧 ADR の
  status 変更で行う。
- **分類**（ここで種別を確定してから移す）：
  - **decision / exclusion** → ADR（`docs/adr/`）。
  - **rule**（日々強制される帰結。層依存・DTO 境界など）→ `docs/rules.md` に残し、
    ADR への backlink を張る。
  - **inventory**（コードに追従して変化する目録）→ `docs/reference/dependencies.md`
    等の生きた文書。ADR にしない。

## 移行対象 ADR 一覧と採番案

種別凡例：`migrate`＝decisions.md から移設 / `new`＝未文書化の潜在決定 /
`exclusion`＝負の決定。★＝最優先の強い新規（明確な代替案＋影響大＋現状ゼロ）。

### 基盤・アーキテクチャ

| # | ADR | 種別 | 出所 |
| --- | --- | --- | --- |
| 0001 | 実用的オニオンアーキテクチャの採用 | migrate | decisions.md:23 |
| 0002 | モジュラモノリス（マイクロサービスは非目標） | new | architecture.md:253, scope.md:46 |
| 0003 | 構造安全をツール＋CI（depguard）で強制 | new | architecture.md:77 |
| 0004 | REST/Worker/Job は駆動アダプタ（分割軸にしない） | new | rest.md:11 |

### API / 永続化

| # | ADR | 種別 | 出所 |
| --- | --- | --- | --- |
| 0005 | OpenAPI-first API 契約 | migrate | decisions.md:63 |
| 0006 | SQL-first ＋ sqlc（SQL-first と sqlc を統合） | migrate | decisions.md:92,119 |
| 0007 | append-only な不変マイグレーション | new | rules.md:94 |
| 0008 | 軽量 CQRS（QueryService 読み取り分離、IF は usecase） | new ★ | rdb/query_service/README.md:8 |
| 0009 | 競合時のトランザクション再試行＋呼び出し側冪等契約 | new ★ | boundary/tx/README.md:16 |
| 0010 | UUIDv7（時間順）識別子戦略 | new | pkg/uuid/README.md:5 |

### HTTP フレームワーク / 基盤

| # | ADR | 種別 | 出所 |
| --- | --- | --- | --- |
| 0011 | Echo を HTTP フレームワークに採用 | migrate | decisions.md:146 |
| 0012 | 優先度順（データ駆動）ミドルウェア連鎖 | new | rest.md:29 |
| 0013 | 外向き HTTP レジリエンス基盤（retry/CB/budget/dual timeout） | new ★ | httpclient/README.md:5 |
| 0014 | egress SSRF / dial ガードのセキュリティ姿勢 | new ★ | http_client_transport.go:18 |

### DI / config / errors

| # | ADR | 種別 | 出所 |
| --- | --- | --- | --- |
| 0015 | Uber Fx を DI＋ライフサイクルに採用 | migrate | decisions.md:172 |
| 0016 | fx を中立 DI 抽象で封じ込め（Registrar/Shutdowner） | new | di/lifecycle/README.md |
| 0017 | 不変・起動時一括・fail-fast な型付き config | new | config/README.md |
| 0018 | プロトコル非依存の集約エラー分類（apperror） | new | apperror/README.md:5 |

### 非同期サブシステム

| # | ADR | 種別 | 出所 |
| --- | --- | --- | --- |
| 0019 | broker 非依存 pull-ack worker scaffold | migrate | decisions.md:199 |
| 0020 | トランザクショナル outbox ＋ at-least-once 配送 | new | design/outbox.md（:5 のリンク切れも解消） |
| 0021 | idempotency サブシステム（単一 tx / scope 必須 / 24h TTL） | new | design/idempotency.md:28 |
| 0022 | one-shot ジョブ実行器（起動毎 fx.App） | new | design/job.md:24 |

### observability / 依存 / 除外

| # | ADR | 種別 | 出所 |
| --- | --- | --- | --- |
| 0023 | config 駆動 observability gating | migrate | decisions.md:272 |
| 0024 | ベンダ中立 OTLP-only エクスポート（Collector 委譲） | new | observability.md:15 |
| 0025 | observability アーキ（2 経路メトリクス / provider 非依存 / 固定サンプリング） | new | design/observability.md:30 |
| 0026 | 単一責務ライブラリ選定ポリシー（＋bridge 例外） | migrate | decisions.md:226 |
| 0027 | コンテナ化ツールチェーン＋mise 固定（再現性） | new | rules.md:281 |
| 0028 | アプリ内レート制限器を持たない | exclusion | out-of-scope.md:11 |
| 0029 | scheduled job の並走制御はスケジューラ委譲 | exclusion | out-of-scope.md:17 |
| 0030 | 汎用 Cache 抽象を持たない | exclusion | out-of-scope.md:49 |

### ADR にしないもの

- **rule（rules.md に残す＋backlink）**：層依存方向 / usecase-境界依存 / 生成コード
  非編集 / ドメイン純粋性 / context 伝播 / DTO 境界変換 / tx 配置 / エラー処理機構
  / comment・doc・testing 規約 / 薄いハンドラ / pkg 相互独立 / 実効的不変性 /
  idempotency claim 機構 / RED メトリクス基数制限。
- **inventory（生きた文書へ）**：直接依存表（→ `docs/reference/dependencies.md`。
  ここで `net/http/otelhttp` / `otel/sdk/log` の欠落も修正）/ ミドルウェア優先度
  定数表（→ REST の設計文書に残す）。

## 粒度方針

**サブシステム/決定単位を基本、独自の代替案を持つものだけ独立 ADR** とし、約 30 本に
収める。例：outbox の retention や MaxAttempts は個別 ADR にせず 0020 の Consequences
に束ねる。より細かく（約 45 本）／より粗く（約 20 本）も選択可。

## フェーズ別作業手順

### Phase 0: 型固定

`docs/adr/` に README（決定ログ＋分類規約＋supersede 運用）/ template（MADR-lite）/
`0000-record-architecture-decisions.md`（メタ ADR）/ 見本 1 本（0001 onion）を作成し、
形式・採番・supersede 運用を確定する。

### Phase 1: 既存 9 件の移設

decisions.md の既存決定（onion / OpenAPI-first / SQL-first＋sqlc / echo / fx / worker /
library policy / o11y gating）を 1:1 で ADR 化。内容は移すだけ。

### Phase 2: 強い新規 4 件

★（0008 CQRS/QueryService、0009 tx 再試行契約、0013 HTTP レジリエンス基盤、
0014 SSRF ガード）を新規作成。現状ゼロで影響が大きいものを優先。

### Phase 3: サブシステム・潜在決定

outbox / idempotency / job / observability 追加、および architecture・rules 由来の
潜在決定（0002-0004, 0007, 0010, 0016-0018, 0025, 0027）を作成。outbox 作成時に
`docs/design/outbox.md:5` のリンク切れを解消。

### Phase 4: 除外（負の ADR）

0028-0030 を作成。out-of-scope.md の該当項目は目録として残しつつ、代替案分析を持つ
3 件のみ ADR へ昇格。

### Phase 5: インベントリ分離・参照貼り替え・撤去

依存表を `docs/reference/dependencies.md` へ移し欠落を修正。全参照を貼り替え、ja を
同期し、`docs/decisions.md` を撤去（または 1 行リダイレクト）。

## 参照貼り替え対象

`docs/decisions.md` は約 20 ファイルから参照されている。

- canonical：README（en/ja）/ rules.md / index.md / maintenance/docs-structure.md
- design：docs/design/README.md / worker.md / outbox.md（および ja 各種）
- portal：docs/portal/guides/（overview / worker-design / outbox-design、en/ja）
- ツール：`.claude/agents/doc-reviewer.md`
- **AGENTS.md**：人間専用ファイル。ここの参照更新は**人手**で行う必要がある。

## リスクと注意点

- **AGENTS.md は AI 編集禁止。** 移行の最終段で人間が参照行を更新する前提。
- **ja 二重管理。** 各 ADR は `docs/ja/adr/` にミラーが必要（`canonicalize-doc`）。
  ファイル数が倍になる。
- **markdownlint。** 見出しに `<...>` 形式を使うと HTML タグ扱いで MD024 誤検知、
  コードフェンスは言語必須。ローカルの mermaid lint は環境依存で失敗する（内容問題
  ではない）。
- **粒度の再議。** サブシステムをまとめすぎると supersede 単位が粗くなる。★の 4 件は
  独立必須。

## 完了条件

- 30 本（粒度合意後の確定数）の ADR が `docs/adr/` に存在し、種別分類が反映されている。
- 依存表が `docs/reference/dependencies.md` に分離され、欠落が修正されている。
- rules.md の該当ルールから対応 ADR へ backlink が張られている。
- `docs/decisions.md` への全参照が貼り替え済み（AGENTS.md は人手更新）。
- ja ミラーが同期され、`docs/decisions.md` が撤去またはリダイレクト化されている。

## 付録: スキャン手順

本計画の候補洗い出しは、リポジトリ 4 面を並列走査して ADR 化候補を
decision / exclusion / rule / inventory に分類する一次スキルで行った。

- **canonical**：decisions.md / architecture.md / rules.md
- **subsystem design**：docs/design/\*.md
- **exclusion**：docs/project/\*.md（out-of-scope 中心）
- **latent**：各パッケージ README＋コードの "why" コメント

再走査が必要なら同手順を反復する。スキル本体はリポジトリには含めない（一次利用）。
