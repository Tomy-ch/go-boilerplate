# inbound

[English](README.md) | 日本語

`inbound` は、HTTP リクエスト（入力）側の **前処理を行うミドルウェアおよびサーバー設定**を DI 経由で提供するレイヤーです。

Binder / Validator / URI 正規化 / IP 抽出を統一的に管理し、API 入口の品質・安全性・一貫性を保証します。

## モジュール一覧

|モジュール|種別|説明|
|---|---|---|
|`URIModule()`|Pre|URI 末尾スラッシュの除去|
|`IPExtractorModule()`|Configurator|クライアント IP 抽出（X-Forwarded-For / 直接）|
|`OpenAPIModule()`|Use|OpenAPI ベースのリクエスト自動バリデーション|

## 注意点

- **Validator（OpenAPI）は UseMiddleware、URI は PreMiddleware** — Priority により正しい順序で実行
- IP Extractor は SecurityConfig / ApplicationConfig に依存 — **本番とローカルで挙動が異なる可能性あり**
- Binder / Validator は handler に影響するため、**controller / domain 層にロジックを漏らさないこと**
- 新しい inbound 機能の追加時は **ServeCfg（Echo 設定）か Pre/UseMiddleware** に分類すること
