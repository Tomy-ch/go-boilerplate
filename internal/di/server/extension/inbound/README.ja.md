# inbound

[English](README.md) | 日本語

`inbound` は、HTTP リクエスト（入力）側の **前処理を行うミドルウェアおよびサーバー設定**を DI 経由で提供するレイヤーです。

Binder / Validator / URI 正規化 / IP 抽出を統一的に管理し、API 入口の品質・安全性・一貫性を保証します。

## モジュール一覧

|モジュール|種別|説明|
|---|---|---|
|`URIModule()`|Pre (priority 1)|URI 末尾スラッシュの除去|
|`TimeoutModule()`|Pre (priority 2)|per-request の deadline budget（`SERVER_REQUEST_TIMEOUT`）|
|`BodyLimitModule()`|Pre (priority 3)|リクエストボディのサイズ上限（`SERVER_BODY_LIMIT_MB`）。OpenAPI 検証がボディを読む前に適用|
|`IPExtractorModule()`|Configurator|クライアント IP 抽出（X-Forwarded-For / 直接）|
|`OpenAPIModule()`|Use|OpenAPI ベースのリクエスト自動バリデーション|

## 注意点

- **Validator（OpenAPI）は UseMiddleware、URI は PreMiddleware** — Priority により正しい順序で実行
- **Timeout は Pre ミドルウェア（priority 2）** — priority が小さいほど先に実行されるため `uri`(=1) の直後(=2)に位置し、per-request の deadline budget が全 `Use`・OpenAPI 検証・ハンドラ・DB を覆うようにする。deadline は `ctx` で伝播し、下流（pgx, `httpclient`）が尊重する。deadline budget の入口として各境界に独立 timeout を置く代わりに単一予算へ集約する
- IP Extractor は SecurityConfig / ApplicationConfig に依存 — **本番とローカルで挙動が異なる可能性あり**
- Binder / Validator は handler に影響するため、**controller / domain 層にロジックを漏らさないこと**
- 新しい inbound 機能の追加時は **ServeCfg（Echo 設定）か Pre/UseMiddleware** に分類すること
