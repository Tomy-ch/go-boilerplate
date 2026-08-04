// Package httpheader は、HTTP ヘッダ名の判定を提供します。
package httpheader

import "strings"

// sensitiveNames は、機微とみなすヘッダ名です（小文字・前後空白なしで保持します）。
var sensitiveNames = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
}

// IsSensitive は、name が資格情報を運ぶヘッダかを返します。
// 判定は大小文字と前後の空白を無視します。ヘッダ名の表記ゆれで判定をすり抜けさせないためです。
func IsSensitive(name string) bool {
	_, sensitive := sensitiveNames[strings.ToLower(strings.TrimSpace(name))]
	return sensitive
}
