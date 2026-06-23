# o11y 基盤リファクタリング設計: トレース正・ログ過剰の解消

- ステータス: 確定（実装着手可 / 未決事項なし）
- 作成日: 2026-06-23
- 最終更新: 2026-06-23
- 対象範囲: `internal/observability`, `internal/logging`, `internal/infrastructure/rdb/driver`(+ loggingdb), 各 repository / query_service / system_query, `internal/di/module/db.go`, testkit

## 1. 背景と課題

現状の o11y 基盤は、span のライフサイクル（開始/終了・latency・親子関係）を **ログとしても出力**している。

- `LayerTracer.Start()` は「本物の OTel span 生成」と「start/end ログ出力」の 2 役を兼ねる（`internal/observability/layer_tracer.go:137-171`、ログ実体は `logSpanEvent` `:159,165`）。
- DB アクセスも同様に、`loggingdb` が Exec/Query/QueryRow ごとに span を切り（`internal/infrastructure/rdb/driver/loggingdb/with_logging.go:40,57,78`）、start/end ログを出す（`:97-100,147-161`）。

結果として 1 リクエストあたり controller/usecase/infra の各層 × start/end、さらに SQL ごとに start/end のログが積み上がり、**ログが過剰**になっている。

これらの span ログは元々 APM 目的で作られたものだが、span 自体は既に OpenTelemetry の本物の span として生成され、`autoexport` 経由でバックエンドへ送出可能（`internal/observability/provider.go:42-62`）。**span のライフサイクルはトレースの信号であり、ログへの二重化はノイズ・コストの両面で負債**になっている。

## 2. 決定（方針）

boilerplate の初期基盤として **「トレースを正とする」** を採用する。

- span の start/end・latency・親子関係は **トレースが唯一の真実**とし、ログには二重化しない。
- バックエンド未選定でも可視性を失わないよう、未設定時は OTel の **console exporter**（`OTEL_TRACES_EXPORTER=console`）で stdout に span を出す運用に寄せる。`autoexport` は既に対応済みのため、手作りの span ログ（`logSpanEvent`）は不要。
- ログは「離散イベント」に絞る: アクセスログ（HTTP req/resp）、エラー、panic、ライフサイクル、そして **SQL のエラー / スロークエリ**。

「ログてんこ盛り」を初期 default にすると下流の全サービスがアンチパターンを継承し、後から剥がすコストが高い。正しい型を初期基盤で固定する。

### 確定した個別判断

| 論点 | 決定 |
| --- | --- |
| layer span の start/end ログ | **両方削除**（span 生成は維持。latency は span が持つ） |
| handler（controller 層）の span | **残す**。「なぜ FindById が呼ばれたか」の意味付けはトレースの span 階層が担う |
| SQL の start ログ | **削除**（span 開始で既出のノイズ） |
| SQL の成功 end ログ（Info） | **削除し、SQL 本文/args/latency は span 属性へ**（semconv） |
| SQL のスロークエリ（Warn） | **ログ維持**（サンプリングで取りこぼさない。バックエンド無しでも効く） |
| SQL のエラー（Error） | **ログ維持**（＋ `span.RecordError`）。サンプリング非対象・grep/alert 可能・離散イベント |
| `loggingdb` パッケージ | **廃止**。計装を pgx の接続層へ移す |
| SQL の計装ポイント | pgx `ConnConfig.Tracer`（`QueryTracer`）に移設 = driver 層で span を取得 |
| span 生成の実装 | **otelpgx（span/semconv）+ 薄い自前ログ tracer（error/slow）を multiTracer で合成** |
| SQL ログの `caller` 欠落 | **許容**。エラー/スロー時は SQL 本文 + trace_id + span 名で追跡 |

## 3. 設計詳細

### 3.1 observability 層: span ログの撤去

- `LayerTracer.startSpan` / `RunWithSpan` から `logSpanEvent` 呼び出しを撤去。span 生成（`StartSpanWithParent`）と endSpan は維持。
- `logSpanEvent` メソッド自体を削除。
- `observability.SpanEventStart` / `SpanEventEnd`（`internal/observability/helper.go:13-15`）を削除。
- `logging.ObservabilityFieldsInput.EventType` フィールドおよび `BuildObservabilityFields` の event_type 連動を撤去（span 由来のログが消えるため）。
- handler / usecase / infra の各 `tracer.Start(ctx)` 呼び出しは **一切変更しない**（span 階層を維持するため）。

### 3.2 driver 層: pgx QueryTracer による計装

- `driver.NewDB`（`internal/infrastructure/rdb/driver/driver.go:32-60`）で `poolCfg.ConnConfig.Tracer` に `QueryTracer` を結線する。
- pgx は pool / tx の両経路で `TraceQueryStart/End` を発火するため、**tx 経由のクエリも自動計装**される（現状の改善点）。必要に応じ `BatchTracer` / `CopyFromTracer` も追加可能（現ラッパーは未カバー）。
- tracer の責務:
  1. クエリ span を生成し、semconv 属性（`db.system` / `db.statement` / `db.namespace` 等）を付与。
  2. `TraceQueryEnd` でエラー時 / スロークエリ時のみログを出力。
  3. エラー時は `span.RecordError` + status 設定。
- `MaskedDBQueryArgs`（PII マスク, `with_logging.go:124`）と `SlowQueryWarnThreshold`（`:152`）のロジックは tracer 内へ移植して維持。
- ログの trace_id/span_id は `TraceQueryEnd` が受け取る ctx の span（`trace.SpanFromContext`）から取得し相関を維持。

### 3.3 loggingdb パッケージの廃止

- `internal/infrastructure/rdb/driver/loggingdb/`（`with_logging.go` / `provider.go` / `mock/`）を削除。
- 各 repository / query_service / system_query は `loggingdb.DBProvider` への依存を解消し、sqlc への DBTX 受け渡しを
  `gen.New(s.db.NewLoggingDB(ctx))` → `gen.New(driver.New(ctx, db))` に置換する。
  - `driver.New(ctx, db)`（`internal/infrastructure/rdb/driver/connection.go:22`）が既に tx/pool の出し分けを担うため、ログ計装が接続層に移れば `DBProvider` 抽象は役目を失う。

## 4. 影響ファイル一覧

### 削除

- `internal/infrastructure/rdb/driver/loggingdb/with_logging.go`
- `internal/infrastructure/rdb/driver/loggingdb/provider.go`
- `internal/infrastructure/rdb/driver/loggingdb/mock/`（生成物）

### 変更（observability / logging）

- `internal/observability/layer_tracer.go` — `logSpanEvent` 撤去、span 生成は維持
- `internal/observability/helper.go` — `SpanEventStart/End` 削除
- `internal/logging/field_builder.go` — `ObservabilityFieldsInput.EventType` 連動撤去（`BuildObservabilityFields`）
- `internal/logging/const.go` — `EventType*` は **維持**（HTTP/SQL error/panic/lifecycle で使用）

### 変更（driver / DI）

- `internal/infrastructure/rdb/driver/driver.go` — `ConnConfig.Tracer` 結線
- 新規: driver 層に `QueryTracer` 実装ファイル（span 属性 + error/slow ログ）
- `internal/di/module/db.go` — `loggingdb.NewLoggingDBProvider` の Provide を撤去、tracer の wiring を追加

### 変更（consumer: NewLoggingDB → driver.New 置換）

- `internal/infrastructure/rdb/repository/user/user_repository.go`
- `internal/infrastructure/rdb/repository/prefecture/prefecture_repository.go`
- `internal/infrastructure/rdb/query_service/user/user_query_service.go`
- `internal/infrastructure/rdb/system_query/idempotency/idempotency_system_query.go`
- `internal/infrastructure/rdb/system_query/healthcheck/health_check_system_query.go`
- `internal/infrastructure/rdb/testkit/test_kit.go`（`NewTestLoggingProvider` の再設計）

### 変更（テスト）

- 上記各パッケージの `*_test.go`、および observability / loggingdb 関連テスト

## 5. 確定した実装方針（旧・未決事項）

### A. span 生成: otelpgx + 薄い自前ログ tracer の合成 ✅

- **otelpgx（`exaring/otelpgx`）に span 生成 + semconv 属性 + status + `RecordError` を委譲**。手書きでの semconv 追従（現状 semconv v1.37.0）を避け、`BatchTracer` / `CopyFromTracer` のカバレッジも無料で得る。
- **error/slow ログ・PII マスク・スロー閾値・zap 連携は薄い自前 `QueryTracer` に残す**（フル制御を維持）。
- pgx の `ConnConfig.Tracer` 枠は 1 つのため、`TraceQueryStart/End` を otelpgx とログ tracer にファンアウトする小さな **multiTracer** を 1 ファイル追加して両者を束ねる。
- 既存プロジェクトは otel sdk / autoexport / contrib / otelecho に全面依存済みのため、otelpgx 追加は依存方針上も自然。

### B. SQL ログの `caller` フィールド欠落: 許容 ✅

- refactor 後、SQL ログは **error / slow 時のみ**発火する。その場面で必要な情報は「どの SQL が・どの trace で失敗/遅延したか」であり、**SQL 本文 + trace_id（→ 親の infra メソッド span に関数名あり）+ span 名**で十分に追跡できる。
- `caller` 維持には呼び出し元情報を ctx に手で通す bespoke plumbing が必要となり、本 refactor の撤去対象と逆行するため採用しない。
- 将来オプション: 必要になれば repository 境界で operation 名（`repository.method`）を ctx に載せ、tracer がログフィールド/span 属性として拾う形に拡張可能（v1 では不要）。

## 6. 実装ステップ（順序）

1. observability: `logSpanEvent` / `SpanEventStart-End` / `ObservabilityFieldsInput.EventType` 撤去（span 生成は維持）。`make gen-api`（mock 再生成）→ `make test`。
2. driver: `QueryTracer` 実装（A の決定に従う）+ `ConnConfig.Tracer` 結線 + `MaskedDBQueryArgs` / `SlowQueryWarnThreshold` 移植。
3. consumer 置換: 各 repo/qs/sq の `NewLoggingDB(ctx)` → `driver.New(ctx, db)`、`di/module/db.go` の wiring 更新、testkit 再設計。
4. `loggingdb` パッケージ削除。
5. テスト更新（Japanese テストケース名・`t.Parallel()`・require/assert 規約遵守）。
6. `make fix` → `make lint` → `make test`（カバレッジが baseline を下回らない / 新規・変更パッケージ 90% 超を確認）。

## 7. 検証

- 単体: observability / driver tracer / 各 repo の分岐網羅。
- 実機: エンドポイントへ curl し、(1) span ログが消えていること、(2) 正常 SQL がログに出ず span 属性に乗ること、(3) エラー / スロークエリのみログに出ること、(4) trace_id が span ログ消失後も error/slow ログに相関すること、を確認。
- console exporter（`OTEL_TRACES_EXPORTER=console`）でバックエンド無し時に span が stdout に出ることを確認。

## 8. リスクとロールバック

- リスク: consumer 置換が 5 パッケージ + DI + testkit に波及。pgx tracer の親 span 連携（ctx 伝播）が正しくないと span 階層が崩れる。
- 緩和: ステップ単位（observability → driver → consumer → 削除）でコミットを分割し、各段で `make test`。
- ロールバック: `loggingdb` 削除は最後のステップに置き、それ以前の段で問題が出たら個別 revert 可能とする。

## 9. 不変条件（このリファクタで壊さないもの）

- handler / usecase / infra の `tracer.Start` による span 階層。
- `logging.EventType*`（HTTP/SQL error/panic/lifecycle のイベント種別）。
- PII マスク（`MaskedDBQueryArgs`）とスロークエリ閾値（`SlowQueryWarnThreshold`）の挙動。
- アクセスログ・エラーハンドラ・recovery・job/server ライフサイクルログ。
