// Package ops は、運用系エンドポイントを識別するためのユーティリティを提供します。
package ops

import "strings"

// IsOpsPath は /metrics, /health, /healthz, /ready, /version のいずれかを運用系エンドポイントとして判定し true を返します。末尾スラッシュは無視されます。
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
