// Package streampath は、SSE stream endpoint の path を判定するユーティリティを提供します。
package streampath

import (
	"net/http"
	"strings"
)

const (
	// prefix は、stream endpoint の path の接頭辞です（OpenAPI の `/v1/streams/{destination}`）。
	prefix = "/v1/streams/"
	// contentType は、SSE として確定した応答の Content-Type です。
	contentType = "text/event-stream"
)

// Is は、path が SSE stream endpoint のものなら true を返します。末尾スラッシュは無視されません
// （destination は path の一部なので、接頭辞より後ろの形は route の判定に任せます）。
func Is(path string) bool {
	return strings.HasPrefix(path, prefix)
}

// IsCommittedStream は、返し終えた応答が SSE として確定したものかを返します。
// path で判定してはいけません — 確定前の拒否は数ミリ秒で終わる普通のレスポンスで、除外の対象では
// ないためです（理由はこのパッケージの README）。
func IsCommittedStream(h http.Header) bool {
	return strings.HasPrefix(h.Get("Content-Type"), contentType)
}
