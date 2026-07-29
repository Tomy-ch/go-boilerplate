# uuid

[English](README.md) | 日本語

`github.com/google/uuid` をラップした UUID 値オブジェクトです。UUIDv7 を生成し、データベース連携をサポートします。

## ラップ対象

`github.com/google/uuid`

## 注意点

- ワイヤ表現は **JSON 文字列**（`"b1d4e0f2-3c5a-4b6d-8e7f-1a2b3c4d5e6f"`）— 非公開配列のみを持つ値オブジェクトのため、既定の構造体符号化では `{}` になり値が失われる。`UnmarshalJSON` は文字列以外の JSON 値を拒否し、`null` はレシーバを変更しない
- `NewTestFromSalt` はテスト専用 — 本番で使用しないこと
- sqlc override により DB の UUID とこの型が統一されるため、手動変換は不要
