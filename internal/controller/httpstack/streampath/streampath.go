// Package streampath は、SSE stream endpoint の path を判定するユーティリティを提供します。
package streampath

import "strings"

// prefix は、stream endpoint の path の接頭辞です（OpenAPI の `/v1/streams/{destination}`）。
const prefix = "/v1/streams/"

// Is は、path が SSE stream endpoint のものなら true を返します。末尾スラッシュは無視されません
// （destination は path の一部なので、接頭辞より後ろの形は route の判定に任せます）。
func Is(path string) bool {
	return strings.HasPrefix(path, prefix)
}
