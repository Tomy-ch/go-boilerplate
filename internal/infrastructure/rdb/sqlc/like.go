// Package sqlc は、LIKE/ILIKE クエリのパターン生成とエスケープのユーティリティを提供します。
package sqlc

import "strings"

// DefaultLikeEscapeChar は、PostgreSQL の LIKE/ILIKE クエリで使用するデフォルトのエスケープ文字です。
const DefaultLikeEscapeChar = "\\"

// WrapPrefixLikePattern は、前方一致（prefix match）用に pattern を作ります。
// escaped は EscapeForLike 済みの文字列を渡すこと。例: "hoge" -> "hoge%"
func WrapPrefixLikePattern(escaped string) string {
	return escaped + "%"
}

// WrapSuffixLikePattern は、後方一致（suffix match）用に pattern を作ります。
// escaped は EscapeForLike 済みの文字列を渡すこと。例: "hoge" -> "%hoge"
func WrapSuffixLikePattern(escaped string) string {
	return "%" + escaped
}

// WrapContainsLikePattern は、部分一致（contains match）用に pattern を作ります。
// escaped は EscapeForLike 済みの文字列を渡すこと。例: "hoge" -> "%hoge%"
func WrapContainsLikePattern(escaped string) string {
	return "%" + escaped + "%"
}

// EscapeForLike は、PostgreSQL の LIKE/ILIKE のために
// % と _ と エスケープ文字自体をエスケープします。
func EscapeForLike(s, esc string) string {
	s = strings.ReplaceAll(s, esc, esc+esc)
	s = strings.ReplaceAll(s, "%", esc+"%")
	s = strings.ReplaceAll(s, "_", esc+"_")

	return s
}
