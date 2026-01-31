package sqlc

import "strings"

const DefaultLikeEscapeChar = "\\"

// WrapPrefixLikePattern は、前方一致（prefix match）用に pattern を作ります。
// 例: "hoge" -> "hoge%"
func WrapPrefixLikePattern(token string) string {
	return token + "%"
}

// WrapSuffixLikePattern は、後方一致（suffix match）用に pattern を作ります。
// 例: "hoge" -> "%hoge"
func WrapSuffixLikePattern(token string) string {
	return "%" + token
}

// WrapContainsLikePattern は、部分一致（contains match）用に pattern を作ります。
// 例: "hoge" -> "%hoge%"
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
