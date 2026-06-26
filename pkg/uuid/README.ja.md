# uuid

[English](README.md) | 日本語

`github.com/google/uuid` をラップした UUID 値オブジェクトです。UUIDv7 を生成し、データベース連携をサポートします。

## ラップ対象

`github.com/google/uuid`

## 注意点

- `NewTestFromSalt` はテスト専用 — 本番で使用しないこと
- sqlc override により DB の UUID とこの型が統一されるため、手動変換は不要
