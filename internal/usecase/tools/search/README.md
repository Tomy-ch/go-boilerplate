# search

English | [日本語](README.ja.md)

Provides utilities for tokenizing search keyword strings — splitting, deduplicating, and limiting token count.

## Role

This package centralizes the one rule for turning a free-text search keyword into a bounded, deduplicated token set. Keeping it here means every search-capable usecase produces tokens with identical semantics instead of each re-implementing splitting and limiting, and the hard cap on length and token count keeps untrusted input from inflating query cost. It is a deterministic, mechanical transformation only — no domain rules and no awareness of how the query layer later consumes the tokens.

## Processing Steps

1. Truncate keyword to `MaxKeywordLength` runes
2. Split by `_` and whitespace (`FieldsFunc` yields non-empty, whitespace-free tokens)
3. Deduplicate (preserve order, first occurrence wins)
4. Limit to `maxTokens`

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
