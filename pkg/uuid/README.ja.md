# uuid

[English](README.md) | 日本語

`github.com/google/uuid` をラップした UUID 値オブジェクトです。UUIDv7 を生成し、データベース連携をサポートします。

## 公開 API

|関数 / メソッド|説明|
|---|---|
|`New()`|UUIDv7 を生成|
|`Parse(s)`|文字列から UUID をパース|
|`NewTestFromSalt(t, salt)`|テスト用の決定的 UUID を生成|
|`String()`|文字列表現を返す|
|`Bytes()`|`[16]byte` を返す|
|`ToPrimitive()`|`google/uuid.UUID` に変換|
|`IsNil()`|ゼロ値か判定|
|`Equal(v)`|UUID の比較|
|`ToPtr()`|UUID のポインタを取得|
|`EqualPtr(v)`|ポインタ経由で比較|
|`Scan(src)` / `Value()`|DB 連携用 `sql.Scanner` / `driver.Valuer` 実装|

## ラップ対象

`github.com/google/uuid`

## 注意点

- `NewTestFromSalt` はテスト専用 — 本番で使用しないこと
- sqlc override により DB の UUID とこの型が統一されるため、手動変換は不要
