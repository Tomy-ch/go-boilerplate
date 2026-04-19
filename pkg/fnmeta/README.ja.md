# fnmeta

[English](README.md) | 日本語

`runtime` から取得したフル関数名を分解し、パッケージ名や関数名を抽出します。

主に `internal/observability` の span 名生成で使用されます。

## 公開 API

|関数|説明|
|---|---|
|`ExtractFunctionName(full)`|フル関数名からメソッド名を抽出|
|`ExtractPackageName(full)`|フル関数名からパッケージ名を抽出|

## ラップ対象

標準ライブラリ `runtime` パッケージ
