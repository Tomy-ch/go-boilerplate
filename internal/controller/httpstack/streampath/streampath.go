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
//
// path だけでは足りません。stream endpoint は確定前に 401 / 400 / 410 / 503 を返すことがあり、
// それらは数ミリ秒で終わる普通のレスポンスです。「接続の長さを request latency として数えると
// 分布が歪む」という除外の理由は確定した接続にしか当てはまらないので、確定したかどうかで見ます。
func IsCommittedStream(h http.Header) bool {
	return strings.HasPrefix(h.Get("Content-Type"), contentType)
}
