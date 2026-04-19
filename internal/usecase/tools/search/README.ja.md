# search

[English](README.md) | 日本語

検索キーワード文字列をトークンに分割し、正規化・重複排除・上限制限を行うユーティリティを提供します。

## 公開 API

|関数 / 定数|説明|
|---|---|
|`ParseSearchTokens(keyword *string, maxTokens int)`|キーワード文字列を正規化してトークン化|
|`DefaultMaxTokens`|デフォルトの最大トークン数（30）|
|`MaxKeywordLength`|キーワードの最大ルーン数（1024）|

## 処理ステップ

1. キーワードを `MaxKeywordLength` ルーンに切り詰め
2. `_` と空白で分割
3. 各トークンの前後空白を除去
4. 空トークンを削除
5. 重複排除（順序保持、先に出現したものを優先）
6. `maxTokens` で制限

## 挙動

- `keyword` が nil または空 → `[]string{}` を返す
- `maxTokens` ≤ 0 → `DefaultMaxTokens`（30）を使用
- 大文字小文字の正規化や Unicode 正規化（NFKC/NFC）は行わない

## 使用例

```go
kw := "foo_bar baz  foo"
tokens := search.ParseSearchTokens(&kw, 10)
// tokens == []string{"foo", "bar", "baz"}
```
