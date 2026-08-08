---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [observability, di]
---

# ADR-0071: オブザーバビリティプロバイダーはライフサイクル非依存（ProviderShutdowner）

English canonical: [0071-lifecycle-independent-provider.md](../../adr/0071-lifecycle-independent-provider.md)

## ステータス

accepted

## 背景

OTel SDK プロバイダー（`TracerProvider`・`MeterProvider`・`LoggerProvider`）にはそれぞれ
グレースフルシャットダウン時にバッファリングされたスパン / メトリクス / ログをフラッシュするために
呼び出さなければならない `Shutdown` メソッドがある。このシャットダウンの登録場所として
DI ライフサイクルフックが自然な選択肢となる。

素朴なアプローチ — プロバイダーコンストラクタが自身の `OnStop` フックを登録する — では、
`internal/observability` が `internal/di/lifecycle` をインポートする必要が生じる。
これは意図した依存関係の方向を逆転させる: `observability` は DI レイヤーがワイヤリングする
基盤であり、DI の内部に依存してはならない。`observability` から `di/lifecycle` への
インポートは循環または層違反の依存関係を生み出す。

## 決定

プロバイダーコンストラクタ（`NewTracerProvider`・`NewMeterProvider`・`NewLoggerProvider`）は
**ライフサイクルに依存しない**。各コンストラクタは具体的な SDK プロバイダー型（例:
`*sdktrace.TracerProvider`・`*sdkmetric.MeterProvider`）を返すため、呼び出し元は具体的な
`Shutdown` メソッドを利用できるが、コンストラクタ自身はシャットダウンフックを登録しない。

`ProviderShutdowner` 型（`shutdown.go`）は DI モジュールが具体的なプロバイダーから
組み立てる OTel 非依存のシャットダウンハンドルを提供する。DI レイヤー
（`internal/di` 内の `hook.RegisterObservabilityShutdownHooks`）がシャットダウン登録を所有する。
これにより `internal/observability` は `internal/di/lifecycle` への依存を持たない。

## 影響

### ポジティブな影響

- `internal/observability` は `internal/di` に依存しない。依存関係の方向が
  クリーンに保たれる（DI がオブザーバビリティをワイヤリングし、逆はない）。
- 具体的なプロバイダー型がプロバイダーパッケージに追加のアダプターなしで DI フックに
  `Shutdown` を直接公開する。
- シャットダウン順序は DI レイヤーによって完全に制御され、他のサブシステムとのフック
  登録を一貫して順序付けできる。

### ネガティブな影響

- プロバイダーの構築とシャットダウン登録が 2 つのパッケージに分かれる（`observability` が
  構築し、`di` が登録する）ため、ライフサイクル全体を理解するには両方を読む必要がある。
- 具体的なプロバイダー型がより狭い `trace.TracerProvider` / `metric.MeterProvider`
  インターフェースではなく戻り値として公開される。インターフェースのみが必要な呼び出し元は
  アダプター関数 `ProvideTracerProvider` / `ProvideMeterProvider` を使用する。

## 検討した代替案

### プロバイダーコンストラクタが自身のライフサイクルフックを登録する

却下: `internal/observability` が `internal/di/lifecycle` をインポートする必要が生じ、
依存関係が逆転し、オブザーバビリティ基盤が DI フレームワークに結合する。

### プロバイダーコンストラクタに注入するライフサイクルインターフェース

却下: DI フレームワークの関心事をオブザーバビリティの API シグネチャに埋め込むことになり、
コンストラクタをライフサイクル抽象に結合させる一方で、インポートサイクルは部分的にしか
解消されない。

### プロセス終了に依存する（明示的な Shutdown なし）

却下: `Shutdown` なしでは、バッファリングされたスパンとメトリクスがグレースフルシャットダウン時に
サイレントにドロップされ、不完全なトレースとメトリクスのギャップが生じる。

## 補足

- 設計上の不変条件（出典）: `docs/design/observability.md` §1「Role theory」、30 行目:
  「プロバイダーはライフサイクルに依存しない — 具体的な SDK プロバイダー（`Shutdown` を
  公開する）を返し、DI フックにシャットダウン登録を委ねることで、`observability` が
  `di/lifecycle` をインポートしない」。
- 実装: `internal/observability/shutdown.go`（`ProviderShutdowner`）、
  `internal/observability/provider.go`（`NewTracerProvider`・`NewMeterProvider`・
  `ProvideTracerProvider`・`ProvideMeterProvider`）、
  `internal/observability/log_provider.go`（`NewLoggerProvider`）。
- 親: [ADR-0067](0067-config-driven-observability-gating.ja.md)。
