---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability]
---

# ADR-0067: ベンダー中立の OTLP 専用エクスポート（バックエンドは Collector に委譲）

English canonical: [0067-vendor-neutral-otlp-export.md](../../adr/0067-vendor-neutral-otlp-export.md)

## ステータス

accepted

## 背景

オブザーバビリティサブシステムはトレース・メトリクス・ログを監視バックエンドへ送信しなければならないが、
アプリケーションバイナリを特定ベンダーの SDK や独自エンドポイントに結合してはならない。
このテンプレートの利用者は異なるバックエンド（Grafana・Datadog・New Relic など）の環境へ
デプロイするため、テンプレートはそれらの選択をまたいで中立でなければならない。
アプリケーションバイナリにベンダー固有のエクスポーターを組み込むと、プロバイダーを切り替える
際にコード変更が必要になり、[ADR-0001](0001-avoid-lock-in.ja.md) のロックイン回避原則に
反する。

## 決定

**ベンダー中立の OTLP 配管のみ**をワイヤリングする。トレース・メトリクス・ログの 3 つの
シグナルすべてが OTLP を唯一のエクスポートトランスポートとして使用する（デフォルトは
`http/protobuf`、`grpc` はオプション）。ベンダー固有のルーティング・パイプライン変換・
バックエンド認証は Collector または Agent サイドカーにのみ存在し、アプリケーションコードや
型付き設定には存在しない。

単一の `OBS_OTLP_ENDPOINT` が 3 つのシグナルで共有される。HTTP の場合、URL にパスが
含まれていなければシグナルごとのパス（`/v1/traces`・`/v1/metrics`・`/v1/logs`）が
自動的に付加される。ベンダー SDK はインポートしない。唯一のエクスポーター依存関係は
OpenTelemetry の OTLP エクスポーターパッケージのみである。

この決定は [ADR-0066](0066-config-driven-observability-gating.ja.md) によってゲーティングされる。
無効なシグナル（エクスポーター値が空または `none`）は OTLP エクスポーターを一切構築しない。

## 影響

### ポジティブな影響

- OTLP 対応バックエンドであれば `OBS_OTLP_ENDPOINT` を Collector に向けるだけで到達でき、
  アプリケーションコードの変更は不要。
- ベンダー SDK はバイナリにコンパイルされない。依存関係の表面は監査可能かつ交換可能な状態を保つ。
- バックエンド固有の関心事（認証・パイプラインルーティング・サンプリングルール）は
  Collector が担い、アプリケーションの関心事と分離される。

### ネガティブな影響

- エクスポートを有効にするすべての環境で Collector または Agent サイドカーが必要になる。
  Collector を介さないベンダー直接エクスポートはこの設計では対応しない。
- 2 つのトランスポートオプション（`http/protobuf` と `grpc`）を維持しなければならない。
  プロトコルの設定誤りは起動時（provider 構築時）のエラー（`errInvalidOTLPProtocol`）となる。

## 検討した代替案

### ベンダーネイティブなエクスポーター（例: Datadog Agent SDK・New Relic SDK）

却下: ベンダー SDK をインポートするとバイナリが 1 つのプロバイダーの API と認証モデルに
結合する。バックエンドの切り替えにはアプリケーション自体のコードと依存関係の変更が必要になり、
[ADR-0001](0001-avoid-lock-in.ja.md) に違反する。

### OTel autoexport（`OTEL_*` 環境変数を直接読む）

却下: autoexport は型付き設定システムの外側で環境変数を読み込み、第二のコントロールプレーンを
生み出す。[ADR-0066](0066-config-driven-observability-gating.ja.md) がこの却下を詳述している。

### ローカル開発用コンソールエクスポーター

却下: no-op フォールバック（無効なシグナルはエクスポーターもゴルーチンも構築しない）で
ローカル開発モードとして十分。コンソールエクスポーターは軽微なメリットしかない第三のコード
パスになってしまう。

## 補足

- 親原則: [ADR-0001](0001-avoid-lock-in.ja.md)（ロックイン回避）。
- 設定駆動ゲーティング: [ADR-0066](0066-config-driven-observability-gating.ja.md)。
- 出典: `docs/design/observability.md` §1「Role theory」、`internal/observability/README.md`
  「Configuration boundary」。
- 実装: `internal/observability/provider.go`（`newSpanExporter` / `newMetricExporter` における
  OTLP 専用エクスポーター構築）。
