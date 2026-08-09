---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, metrics]
---

# ADR-0071: メトリクスは 2 経路を通る — OTLP プッシュと Prometheus スクレイプ

English canonical: [0071-dual-path-metrics.md](../../adr/0071-dual-path-metrics.md)

## ステータス

accepted

## 背景

オブザーバビリティサブシステムは収集の形状が根本的に異なる 2 種類のメトリクスを生成する。

- **操作サブシステムメトリクス**（outbox ラグ・worker RED・べき等性結果・DB スパン・Go ランタイム）は
  イベント駆動: アプリケーションが処理を進めるにつれて蓄積され、SDK によるプッシュスケジュールで
  送信するのが最適。
- **プロセス / ブローカー識別メトリクス**（ビルド情報・ブローカーキュー深度）はプル指向:
  ビルドの識別情報はワイヤリング時に一度解決されて以降変化しない。キュー深度は定期的に
  バッファリングしてプッシュするより、スクレイプのたびにオンデマンドで読むのが最適。

両カテゴリを同じ OTLP プッシュ経路に通すと、プル指向のメトリクスを無理なプッシュパターンに
押し込むことになる（純粋に計装目的のためだけにバックグラウンドゴルーチンでポーリングして
OTLP へプッシュする）か、Prometheus スクレイプエンドポイントを維持するために OTLP メトリクスを
完全に無効化しなければならなくなる。

## 決定

メトリクスを**意図的かつ独立した 2 経路**で公開する。

| 経路 | 計装 | プロセスからの出力方法 |
| --- | --- | --- |
| **OTLP プッシュ** | `outbox` / `worker` / `idempotency` / `httpclient` OTel メーター + Go ランタイム + `otelpgx` DB メトリクス | `MeterProvider` の `PeriodicReader` が Collector へプッシュ。`MetricsEnabled()` のときのみ有効 |
| **Prometheus スクレイプ** | `app_build_info`（buildinfo）・`worker_queue_depth`（queue） | デフォルトの Prometheus レジストリに登録され、`promhttp` 経由で `/metrics` に提供。`OBS_METRICS_EXPORTER` に依存しない |

スクレイプ経路は OTLP シグナルが有効かどうかに関わらず常に存在する。2 つの経路は共存する。
監視スタックに応じて、両方を有効にする・スクレイプエンドポイントのみ・OTLP プッシュのみという
構成が可能。

## 影響

### ポジティブな影響

- プル指向のメトリクス（ビルド識別情報・ライブキュー深度）は固定間隔でプッシュされるのではなく、
  オンデマンドで効率的に収集される。
- スクレイプエンドポイントは Collector なしでも `OBS_METRICS_EXPORTER` を有効にしなくても
  動作するため、軽量 / ローカルデプロイメントでもクエリできる。
- 各経路は独立して運用される: OTLP プッシュ経路とスクレイプ経路は状態を共有せず、
  互いに結合を導入しない。

### ネガティブな影響

- 全メトリクスの全体像を把握するには、オペレーターが両経路を認識しなければならない。
  Prometheus スクレイプエンドポイントと OTLP Collector は設定・監視が必要な 2 つの
  異なる取り込みポイントとなる。
- 一方の経路に属するメトリクスは、Collector 側のブリッジ（例: Prometheus remote write）なしに
  もう一方の経路で簡単にクエリすることはできない。

## 検討した代替案

### OTLP プッシュのみ（Prometheus コレクターを OTel MeterProvider にブリッジ）

却下: キュー深度はライブプル（スクレイプごとにブローカー状態をサンプリングするのが
`worker.QueueStatsProvider` の意図するモデル）であり、プッシュパイプラインへバッファリングすると
古い値の読み取りウィンドウと純粋に計装目的のバックグラウンドゴルーチンが生まれる。

### Prometheus スクレイプのみ

却下: OTel メーター計装は SDK パイプライン（バッチエクスポート・エグゼンプラー・リソース帰属）と
ネイティブに統合されるが、Prometheus 計装はそうではない。

### Collector 内の Prometheus-to-OTLP ブリッジを通じた統合

ここでは採用しない: これは Collector 側の関心事であり、アプリケーションは Collector の
設定を規定しない。

## 補足

- 出典: `docs/design/observability.md` §3.2「Two metric exit paths」、146–153 行目のテーブル。
- 親: [ADR-0068](0068-config-driven-observability-gating.ja.md)（設定駆動ゲーティング）。
- 実装: `internal/observability/metrics/buildinfo/`（buildinfo コレクター）、
  `internal/observability/metrics/queue/`（キュー深度コレクター）、
  `outbox_metrics.go`・`worker_metrics.go`・`idempotency_metrics.go`・
  `http_client_metrics.go` 内の OTel メーター計装。
