# fnmeta

[English](README.md) | 日本語

`runtime` から取得したフル関数名を分解し、パッケージ名や関数名を抽出します。

関数のランタイムフルネームから、短く人間可読な識別子（span 名やログ名など）を導出する用途に使います。

## ラップ対象

標準ライブラリ `strings` パッケージ。`runtime` が生成するフル関数名文字列（例: `runtime.FuncForPC(...).Name()`）を解析しますが、`runtime` 自体は import しません（フル関数名は呼び出し側が取得します）。
