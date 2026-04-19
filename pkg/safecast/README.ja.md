# safecast

[English](README.md) | 日本語

オーバーフローを検出する安全な型変換を提供します。

## 公開 API

|関数 / 変数|説明|
|---|---|
|`UintToInt(x uint) (int, error)`|`uint` → `int` の安全な変換|
|`ErrOverflow`|オーバーフロー時に返されるエラー|

## 注意点

値が `math.MaxInt` を超える場合に `ErrOverflow` を返します。
