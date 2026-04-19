# xerrors

[English](README.md) | 日本語

`github.com/cockroachdb/errors` をラップし、スタックトレース付きのエラー操作を提供します。

## 公開 API

|関数|説明|
|---|---|
|`New(msg)`|スタックトレース付きの新しいエラーを生成|
|`Wrap(err, msg)`|既存エラーをメッセージとスタックトレース付きでラップ|
|`Is(err, target)`|エラーの同一性を判定（ラップチェーン対応）|
|`As(err, target)`|エラーの型アサーション（ラップチェーン対応）|
|`StackTrace(err)`|フォーマット済みスタックトレース文字列を取得|

## ラップ対象

`github.com/cockroachdb/errors`

## 注意点

- このパッケージ経由で生成されたエラーはすべてスタックトレースを持つ
- 一貫性のため `errors.Is` / `errors.As` ではなく本パッケージの `Is` / `As` を使用すること
