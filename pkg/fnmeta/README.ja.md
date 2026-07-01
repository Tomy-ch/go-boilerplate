# fnmeta

[English](README.md) | 日本語

`runtime` から取得したフル関数名を分解し、パッケージ名や関数名を抽出します。

主に `internal/observability` の span 名生成で使用されます。

## ラップ対象

標準ライブラリ `strings` パッケージ。`runtime` が生成するフル関数名文字列（例: `runtime.FuncForPC(...).Name()`）を解析しますが、`runtime` 自体は import しません（フル関数名は呼び出し側が取得します）。
