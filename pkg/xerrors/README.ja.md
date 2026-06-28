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
- apperror センチネルを元エラーに付与する場合は、`Wrap(sentinel, err.Error())` で潰さず
  `Join(sentinel, err)`（文脈が要る場合は `Join(sentinel, Wrap(err, "文脈"))`）を優先すること。
  これにより元エラーの型・スタックが chain に残り `Is` / `As` で辿れる。
  - 例外: 元エラーが機密情報（クエリ・userinfo を含む URL など）を含みうる場合は `Join` しないこと。
    メッセージを redact してから `Wrap(sentinel, <redact済み文字列>)` し、生のエラーを伝播させない。
