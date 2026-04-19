# decoration

[English](README.md) | 日本語

`decoration` は、サーバー起動時の **視覚的・初期表示に関する機能**（バナー表示 / デフォルトポート設定）を DI 経由で提供するレイヤーです。

## モジュール一覧

|モジュール|種別|説明|
|---|---|---|
|`BannerModule()`|Configurator|環境に応じて Echo バナー表示を制御|
|`DefaultPortModule()`|Configurator|環境に応じてポート番号表示を制御|

## 注意点

- バナー表示は UI 的な装飾のため、**ビジネスロジック（domain/usecase）に依存させないこと**
- `DefaultPort` は `ApplicationConfig` に依存するため、環境差異に注意
- decoration はあくまで「サーバー起動時の補助」であり、本質的なミドルウェアと混在させないこと
