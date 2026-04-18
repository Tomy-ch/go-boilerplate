# search

English | [日本語](README.ja.md)

Provides utilities for tokenizing search keyword strings — splitting, normalizing, deduplicating, and limiting token count.

## Public API

|Function / Constant|Description|
|---|---|
|`ParseSearchTokens(keyword *string, maxTokens int)`|Tokenize keyword string with normalization|
|`DefaultMaxTokens`|Default max token count (30)|
|`MaxKeywordLength`|Maximum keyword length in runes (1024)|

## Processing Steps

1. Truncate keyword to `MaxKeywordLength` runes
2. Split by `_` and whitespace
3. Trim each token
4. Drop empty tokens
5. Deduplicate (preserve order, first occurrence wins)
6. Limit to `maxTokens`

## Behavior

- `keyword` nil or empty → returns `[]string{}`
- `maxTokens` ≤ 0 → uses `DefaultMaxTokens` (30)
- No case normalization or Unicode normalization (NFKC/NFC) is performed

## Usage

```go
kw := "foo_bar baz  foo"
tokens := search.ParseSearchTokens(&kw, 10)
// tokens == []string{"foo", "bar", "baz"}
```
