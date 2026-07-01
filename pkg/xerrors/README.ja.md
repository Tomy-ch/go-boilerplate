# xerrors

[English](README.md) | 日本語

`github.com/cockroachdb/errors` をラップし、スタックトレース付きのエラー操作を提供します。

## ラップ対象

`github.com/cockroachdb/errors`

## 注意点

- このパッケージ経由で生成されたエラーはすべてスタックトレースを持つ
- `StackTrace(err)` は `%+v` 表現（スタックが付いていればそれも含む）を返す（`err` が `nil` なら空文字列）
- 一貫性のため `errors.Is` / `errors.As` ではなく本パッケージの `Is` / `As` を使用すること
- エラーの生成・ラップには `fmt.Errorf` ではなく `New` / `Wrap` / `Join` を使用すること
  （`Join` は複数エラーの結合）。メッセージに値を埋め込む場合は `fmt.Sprintf` で整形してから `New` / `Wrap` に渡すこと。
  `fmt.Errorf` は `forbidigo` で禁止されています。
- apperror センチネルを元エラーに付与する場合は、`Wrap(sentinel, err.Error())` で潰さず
  `Join(sentinel, err)`（文脈が要る場合は `Join(sentinel, Wrap(err, "文脈"))`）を優先すること。
  これにより元エラーの型・スタックが chain に残り `Is` / `As` で辿れる。
  - 例外: 元エラーが機密情報（クエリ・userinfo を含む URL など）を含みうる場合は `Join` しないこと。
    メッセージを redact してから `Wrap(sentinel, <redact済み文字列>)` し、生のエラーを伝播させない。
  - 注意（意図的な型消去）: `Wrap(sentinel, err.Error())` は元エラーの型を chain から**意図的に消す**ため、
    下流の `Is` / `As` はその型に到達できなくなる。既存の正規化器を `Wrap` から `Join` へ変える前に、
    結果を検査する述語をすべて確認すること。元の型に**マッチしないこと**に依存している述語があると、
    `Join` で型が再露出して挙動が黙って変わる（例: `*pgconn.PgError` の SQLSTATE を見る tx リトライ述語）。
    型消去が意図的な場合は `Wrap` を維持すること。
