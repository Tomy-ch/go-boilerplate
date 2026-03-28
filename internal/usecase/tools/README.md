# tools パッケージ

`tools` パッケージは、ユースケース層で再利用される小さなユーティリティ群を収めています。現在は主に `paging`（ページングの共通処理）と `search`（検索キーワードのトークン化）を提供します。

## 提供する主な機能

- `paging` サブパッケージ
  - `NewPagingFrom1Based(page, perPage *int) (*Paging, error)`
    - 1 ベースのページ番号と 1 ページあたり件数から、内部で使用する `limit`（取得上限）と `offset`（オフセット）を計算するヘルパーを返します。
    - 入力のバリデーション（ページ番号の上限チェックや per-page の最大値制限）を含みます。
  - `Paging` 型は `Limit()` / `Offset()` メソッドを提供します。

- `search` サブパッケージ
  - `ParseSearchTokens(keyword *string, maxTokens int) []string`
    - 入力キーワードを分割（スペースまたはアンダースコアで分割）、前後トリム、空要素除去、重複排除、上限トークン数の制限を行います。
  - トークン化の内部処理には `splitIntoTerms` / `trimAndDropEmpty` / `dedupePreserveOrder` / `limit` を用います。

## 使い方（例）

1) ページングの生成例

```go
pg, err := paging.NewPagingFrom1Based(ptr.To(1), ptr.To(20))
if err != nil {
  // エラーハンドリング
}
0limit := pg.Limit()
o00ffset := pg.Offset()
// これらをリポジトリのクエリに渡す
```

1) 検索キーワードのトークン化

```go
kw := "foo_bar baz  foo"
tokens := search.ParseSearchTokens(&kw, 10)
// tokens == []string{"foo", "bar", "baz"}
```

## 注意点

- `NewPagingFrom1Based` はページ番号や件数が不正（非常に大きい等）な場合に `apperror.ErrInvalidArgument` を返す可能性があります。呼び出し側で適切にハンドリングしてください。
- `ParseSearchTokens` は入力文字列が `nil` または空の場合、空スライスを返します。トークン数の上限は第二引数で制御できます（0 以下の場合はデフォルト値が適用されます）。
