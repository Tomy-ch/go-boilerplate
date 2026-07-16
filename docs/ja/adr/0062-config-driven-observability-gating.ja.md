---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, config]
---

# ADR-0061: 設定駆動によるオブザーバビリティゲーティング

English canonical: [0061-config-driven-observability-gating.md](../../adr/0061-config-driven-observability-gating.md)

## ステータス

accepted

## 背景

オブザーバビリティ（トレース / メトリクス / ログ）は環境ごとに切り替え可能でなければならず、
軽量環境では OpenTelemetry プロバイダー・エクスポーター・計装ブリッジを**一切**構築しないようにする。
アプリの背後で別のコントロールプレーン（spec 標準の `OTEL_*` 環境変数を autoexport で読む形）が
読み込まれると、プロジェクトの型付き設定から切り離された第二の真実の源が生まれてしまう。

## 決定

オブザーバビリティを単一の型付き**設定駆動**スイッチにする。

- エクスポーター設定は型付き `OBS_*` 設定サブシステム（`OBS_TRACES_EXPORTER` /
  `OBS_METRICS_EXPORTER` / `OBS_LOGS_EXPORTER` / `OBS_OTLP_ENDPOINT` /
  `OBS_OTLP_PROTOCOL`）に置く。autoexport が読む `OTEL_*` 環境変数には置かない。
- **専用の有効化フラグは設けない**。オブザーバビリティは、3 つのエクスポーター設定のいずれかが
  空でも `none` でもない値を持つ場合に有効と*導出*される。エクスポーター設定から導出する設計は
  意図的なものである。Enable フラグだけが有効でエクスポーターが機能していない状態では意味をなさず、
  エクスポーター設定を意識せざるを得ない状況へ追い込む設計にしている。
- ゲーティングは**構築時**に適用される。無効なシグナルはエクスポーター / バッチャー /
  リーダー / ランタイムコレクターを構築しない（ネットワーク接続もゴルーチンもなし）。
  Echo の otelecho ミドルウェアはパススルーに縮退し、otelzap のログコアはロガーに
  Tee されない。SDK プロバイダーシェルは存在したまま（安価で不活性）— これはランタイムの
  無効化であり、ビルド時の除去ではない。
- 同じ config 駆動ゲートは**ログごとの trace 相関**も統制する。`trace_id` / `span_id` は
  `Logger` 自身が、ctx が有効な span を持つ各ログ行へ注入し（ctx-native API — `Info(ctx, msg, ...)`）、
  呼び出し側が付与することはない。ゲート（observability が有効**かつ** ctx が有効な span を持つこと）は、
  `Logger` に DI 注入される単一の `observability.NewTraceExtractor(obsCfg)` クロージャに
  集約される（`logging.TraceExtractor` として注入）。`logging` はこの抽象のみに依存し
  `observability` を import しないため、外層から内側へ向かう依存方向を反転させることなく、
  ゲートを config 駆動のまま保てる（[ADR-0003](0003-interface-based-decoupling.ja.md) 参照）。

## 影響

### ポジティブな影響

- 他のすべてのサブシステムと一貫した、単一の型付き真実の源。第二のコントロールプレーンがない。
- 軽量環境ではオブザーバビリティのコストをゼロにできる（エクスポーター・リーダー・
  コレクター・リクエストごとのスパンなし）。
- ポータビリティが保たれる: OTLP 対応バックエンドであれば `OBS_*_EXPORTER=otlp` とエンドポイントを
  指定するだけでよい。
- `trace_id` / `span_id` のゲートは 1 箇所（注入される extractor）にのみ存在する。呼び出し側は
  `trace_id` / `span_id` ではなく `ctx` を渡す。唯一の例外は `parent_span_id` で、ctx から
  導出できないため `BuildSQLEndFields` 内で `obsCfg.Enabled()` により直接ゲートされる。

### ネガティブな影響

- 「有効」は明示ではなく導出される。オペレーターは状態を知るためにエクスポーター設定を
  確認しなければならない（これは削除された `OBSERVABILITY_ENABLED` フラグの意図的な代替）。
- 計装の依存関係はリンクされたまま残る（ランタイムで無効化されるが、ビルドから除去されるわけではない）。

## 検討した代替案

### `OTEL_*` + autoexport を維持する

却下: エクスポーター設定が型付き設定に存在しなくなり、環境から直接読まれる第二の
真実の源が残る。

### エクスポーター設定と並行した専用の `OBSERVABILITY_ENABLED` フラグ

却下: 「いずれかのエクスポーターが設定されているか」と冗長であり、矛盾した状態
（`ENABLED=true` でエクスポーターなし）を引き起こしやすい。これにより以前は切り離されていた
2 つのコントロールプレーンが統合される。

### otel / ブリッジ依存関係のビルドタグによる除去

現時点では却下: ランタイムの無効化で軽量化の目標を達成している。ビルド時除去は
ホットパスにワイヤリングされた計装（otelecho / otelpgx）に対して 2 つのワイヤリング
バリアントを追加することになるが、現在その要件はない。

## 補足

- ベンダー中立の OTLP 専用エクスポートと公式 semconv に関する方針は別途記録されている。
- 設計参照: `docs/design/observability.md`。ctx-native な `Logger` と、ログごとの trace ゲートを担う注入型 `TraceExtractor` は `internal/logging/README.md` に記載。
- 移行元: `docs/decisions.md`（§「Why config-driven observability gating」）。
