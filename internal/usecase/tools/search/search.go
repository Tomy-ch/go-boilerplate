// Package search は、検索関連のユースケースをサポートするユーティリティを提供します。
package search

import (
	"strings"
	"unicode"
)

const (
	// DefaultMaxTokens は、キーワードをトークンに分割する際のデフォルトのトークン数を表します。
	DefaultMaxTokens = 30
	// MaxKeywordLength は、キーワードの最大長を表します。
	MaxKeywordLength = 1024
)

// ParseSearchTokens は、キーワード文字列をトークンに分割し、正規化、重複排除、上限設定を行います。
func ParseSearchTokens(keyword *string, maxTokens int) []string {
	if keyword == nil || *keyword == "" {
		return []string{}
	}
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}

	k := *keyword
	rs := []rune(k)
	if len(rs) > MaxKeywordLength {
		k = string(rs[:MaxKeywordLength])
	}

	raw := splitIntoTerms(k)
	normalised := trimAndDropEmpty(raw)
	unique := dedupePreserveOrder(normalised)

	return limit(unique, maxTokens)
}

// splitIntoTerms は、キーワード文字列を '_' または空白で分割します。
func splitIntoTerms(keyword string) []string {
	return strings.FieldsFunc(keyword, func(r rune) bool {
		return r == '_' || unicode.IsSpace(r)
	})
}

// trimAndDropEmpty は、配列の各要素の前後の空白を削除し、空文字列の要素を排除します。
func trimAndDropEmpty(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, t := range ss {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

// dedupePreserveOrder は、配列の要素の重複を排除し、元の順序を保持します。
func dedupePreserveOrder(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))

	for _, t := range ss {
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// limit は、配列の要素数を上限までに制限します。
func limit(ss []string, maxVal int) []string {
	if len(ss) <= maxVal {
		return ss
	}
	return ss[:maxVal]
}
