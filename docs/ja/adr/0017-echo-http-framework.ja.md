---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [http, framework]
---

# ADR-0017: HTTP フレームワークとして Echo を採用する

English canonical: [0017-echo-http-framework.md](../../adr/0017-echo-http-framework.md)

## ステータス

accepted

## 背景

このテンプレートはルーティングとミドルウェアのための HTTP フレームワークを必要としている。軽量で予測可能であり、標準ライブラリへの抽象化が低く、保守性と構造的安全性を生の機能の豊富さより重視するという設計目標に一致していることが求められる。

Echo v4 はセキュリティ修正とバグ修正のみを受ける状態であり、依存の脆弱性を CI でゲートするテンプレートが EOL の HTTP フレームワークを抱え続けることはできない。Echo v5 が保守される系列であり、周辺エコシステム（OpenAPI バリデーション・OpenTelemetry 計装）も別モジュールとして v5 対応を提供している。

## 決定

HTTP ルーティングとミドルウェアに **Echo v5**（`labstack/echo/v5`）を採用し、HTTP スタックが依存するエコシステムの 2 モジュールも併せて採用する。

| 関心事 | モジュール |
| --- | --- |
| ルーティング / ミドルウェア | `github.com/labstack/echo/v5` |
| OpenAPI リクエストバリデーション | `github.com/oapi-codegen/echo-v5-middleware` |
| OpenTelemetry 計装 | `github.com/labstack/echo-opentelemetry` |

`echo-opentelemetry` は pre-1.0 だが、Echo 自身の「official middleware repositories」に掲載され Echo のメンテナが保守している。OTel contrib の Echo 計装は v4 専用であり、そのメンテナは v5 対応を見送ってこのモジュールを案内している。版番号は新規パッケージの保守的な採番であって API の不安定さを示すものではなく、依存する OTel のバージョンは本テンプレートが既に固定しているものと一致する。

**サーバーのライフサイクルはフレームワークではなくテンプレートが持つ。** Echo v5 は `Echo` からサーバーのフィールドと起動・停止メソッドを削除し `StartConfig` へ集約した。そのモデルは「context がキャンセルされるまでブロックし、その後は自前の graceful timeout 内で停止する」というもので、起動と停止のフックを分離する DI コンテナと噛み合わない。そのためテンプレートは `Echo` をハンドラとする `http.Server` を自前で構築する。リスナは起動フックで開くため bind 失敗は即座に起動を中断でき、`Shutdown` は停止フックの context が駆動する。`Echo.Server` が無くなった今、リクエストのタイムアウトの置き場も `http.Server` である。

**span のエラー詳細は semantic conventions に従う。** v4 の計装はエラー本文を非標準の `echo.error` 属性として span に載せていたが、v5 の計装は代わりに `error.type` を記録する。エラー本文は 5xx では span status の description に入り、トレースと相関するアプリケーションログにも残る。旧属性を復元するフックは追加しない。

## 影響

### ポジティブな影響

- 優先順位付きミドルウェアチェーンが構築するシンプルで明確なミドルウェア構造。
- 標準の `net/http` モデルへの低い抽象化。
- 汎用バックエンドとして十分なパフォーマンス。
- span のクライアントアドレスが生の転送ヘッダではなく設定済みの `IPExtractor` 由来になるため、偽装ヘッダがテレメトリへ届かなくなる。

### ネガティブな影響

- コントローラー層にフレームワーク依存が生じる。ハンドラーは `*echo.Context` に結合されるが、コントローラー境界に封じ込められる（内側に漏れることはない）。
- OpenTelemetry 計装はメンテナの少ない pre-1.0 モジュールである。到達点はミドルウェアスロット 1 箇所に限られ、トレースを無効化すれば統合全体が素通しミドルウェアへ落ちるため、そこでの障害はサーバーを落とさずテレメトリの劣化に留まる。

## 検討した代替案

### Gin

非常に似たフレームワークだが、Echo のミドルウェア構造の方がわずかにシンプルである。

### Chi

優れたルーターだが、Echo はすぐに使えるフレームワーク機能をより完全なセットで提供する。

### Echo 固有の計装ではなく `otelhttp` で包む

`otelhttp` は既に依存に入っており、Echo の外側に素の `http.Handler` として置ける。しかしルートテンプレートを参照できないため span 名が生パスへ退化し、メトリクスのカーディナリティもそれに追随する。既定ではなく避難先として保持する。

## 補足

- ミドルウェアチェーンの設計（優先順位付き・データ駆動）と HTTP スタックの層構成は別途記録している（[ADR ログ](README.ja.md) の HTTP 層 ADR を参照）。
- `docs/decisions.md`（§ "Why Echo"）から移行。
