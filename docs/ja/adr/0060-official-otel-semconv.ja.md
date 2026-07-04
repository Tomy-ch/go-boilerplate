---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, exclusion, setup-review]
---

# ADR-0060: 公式 OpenTelemetry セマンティック規約のみを使用し、カスタム semconv の発明や型付き設定へのベンダーキー追加は行わない

English canonical: [0060-official-otel-semconv.md](../../adr/0060-official-otel-semconv.md)

## ステータス

accepted

## 背景

OpenTelemetry はバージョン管理されたセマンティック規約パッケージ（`semconv`）を公開しており、
サービス ID・リソース属性・テレメトリシグナルの標準的な属性名を定義している。
ベンダー固有のキー（例: Datadog タグ・Grafana ラベル）やプロジェクト固有の属性名を
型付き `ObservabilityConfig` に直接追加して、ベンダーのルーティングやエンリッチメントを
型付き設定レイヤーから駆動したいという要求が繰り返し発生している。
同様に、公式スキーマが必要なラベルをまだカバーしていない場合に非標準のリソース属性を
追加したいという圧力もある。

## 決定

カスタムのセマンティック規約キーを発明したり、ベンダー固有の OTLP 属性キーを
型付き `ObservabilityConfig` に配置したりすることを意図的に**行わない**。

`NewResource` 内のすべてのリソース属性は公式の semconv パッケージ
（`go.opentelemetry.io/otel/semconv/v1.37.0`）のみを使用する:
`semconv.ServiceName`・`semconv.DeploymentEnvironmentName`・`semconv.ServiceVersion`。
直接対応する semconv が存在しない 2 つのビルド時 ID フィールド（`service.revision`・
`service.build_date`）は、ベンダー固有のキーではなく、安定した非ベンダーキー名での
プレーンな `attribute.String` 呼び出しで表現する。ベンダー固有のエンリッチメントと
ルーティングはアプリケーション設定ではなく Collector に置く。

セットアップレビュアーは、初期プロジェクトセットアップ時に導入される新しいリソース属性や
設定フィールドがこの境界を引き続き守っていることを確認しなければならない。

## 影響

### ポジティブな影響

- リソース属性は任意の OTLP 対応バックエンドで移植可能なまま保たれる。
  アプリケーションコードにベンダー固有のマッピングが不要。
- 型付き設定に OTLP 固有またはベンダー固有のキーが入らず、`ObservabilityConfig` が
  読みやすく監査しやすい状態を保つ。
- semconv バージョンのアップグレードは単一のインポートパス変更であり、
  カスタム / 公式が混在するキー名前空間を横断して検索する必要がない。

### ネガティブな影響

- 公式の semconv が必要な属性をまだカバーしていない場合、チームは標準の対応を待つか、
  明確に名前空間化されたベンダーでない名前のプレーンな `attribute.String` キーを
  受け入れる必要がある。便宜的なベンダーキーはショートカットとして使えない。

## 検討した代替案

### ベンダー固有キーを `ObservabilityConfig` に追加する

却下: バックエンドルーティングの決定が型付き設定レイヤーに埋め込まれ、特定ベンダーへの
結合が生まれ、[ADR-0059](0059-vendor-neutral-otlp-export.ja.md) のベンダー中立の立場が
損なわれる。ベンダーエンリッチメントは Collector パイプラインに属する。

### プロジェクトローカルな semconv 風キーを発明する

却下: 並行するキー名前空間を発明すると、公式の semconv が最終的に同じ概念をカバーした
ときに属性名のズレが生じ、バックエンド固有の再マッピングが必要な非移植的なテレメトリが
生成される。

## 補足

- 補完する決定: [ADR-0059](0059-vendor-neutral-otlp-export.ja.md)（ベンダー中立エクスポート）。
- 親ゲーティング: [ADR-0058](0058-config-driven-observability-gating.ja.md)。
- 実装根拠: `internal/observability/provider.go` 23 行目（semconv インポート）、
  58–60 行目（`NewResource` 内の `semconv.ServiceName` / `semconv.DeploymentEnvironmentName` /
  `semconv.ServiceVersion`）; `internal/observability/README.md` 46–51 行目
  （「OTLP 固有のキーを型付き設定に漏らさない」）。
- セットアップレビュアーの対応: プロジェクトスキャフォールドのカスタマイズ時に、
  `ObservabilityConfig` の新規フィールドがベンダー固有の OTLP キー名を持たないことを確認する。
