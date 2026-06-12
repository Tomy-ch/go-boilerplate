// Package ops は、OpenAPIのパス定義に関連する操作を提供します。
package ops

import "strings"

// IsOpsPath は運用系エンドポイント（ops endpoints）かどうかを判定します。
func IsOpsPath(path string) bool {
	path = strings.TrimRight(path, "/")

	switch path {
	case "/metrics",
		"/health",
		"/healthz",
		"/ready",
		"/version":
		return true
	default:
		return false
	}
}
