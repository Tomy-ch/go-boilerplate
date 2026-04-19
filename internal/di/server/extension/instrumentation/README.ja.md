# instrumentation

[English](README.md) | 日本語

`instrumentation` は、HTTP レイヤーにおける **可観測性（Observability）と ID 付与（RequestID）を提供するミドルウェア DI モジュール群**をまとめたディレクトリです。

Tracing / Logging / Metrics の基盤となる **リクエスト識別子の生成**と**トレース連携ミドルウェア**を提供します。

## モジュール一覧

|モジュール|種別|Priority|説明|
|---|---|---|---|
|`RequestIDModule()`|Use|1|リクエスト単位の一意な ID を生成|
|`LoggingModule()`|Use|—|HTTP リクエスト / レスポンスの構造化ログ|
|`ObservabilityModule()`|Use|2|OpenTelemetry トレーシング統合|

## Priority 順序

RequestID（Priority 1）→ Observability（Priority 2）の順で、**ID 付与 → トレース開始**となるよう設計されています。

## 注意点

- RequestID と Observability は **UseMiddleware として Priority 付きで適用**
- Observability は `ApplicationConfig` に依存 — **本番 / 非本番で挙動が変わる可能性あり**
- 可観測性の責務は controller 層まで — **domain / usecase に漏らさないこと**
- ミドルウェア追加や Priority 変更時は、他の UseMiddleware との衝突に注意
