# xerrors

[English](README.md) | 日本語

`github.com/cockroachdb/errors` をラップし、スタックトレース付きのエラー操作を提供します。

## ラップ対象

`github.com/cockroachdb/errors`

## 注意点

- このパッケージ経由で生成されたエラーはすべてスタックトレースを持つ
- 一貫性のため `errors.Is` / `errors.As` ではなく本パッケージの `Is` / `As` を使用すること
- エラーの生成・ラップには `fmt.Errorf` ではなく `New` / `Wrap` / `Join` を使用すること
  （`Join` は複数エラーの結合）。メッセージに値を埋め込む場合は `fmt.Sprintf` で整形してから `New` / `Wrap` に渡すこと。
  `fmt.Errorf` は `forbidigo` で禁止されています。
