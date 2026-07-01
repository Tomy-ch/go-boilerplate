package redmetrics

import (
	"net/http"
	"strconv"
)

// statusClassUnknown は、分類できない status code に対する status_class label の値です。
const statusClassUnknown = "unknown"

// statusCodeUnknown は、確定できない status code に対する status_code label の値です。
const statusCodeUnknown = "unknown"

// normalizeStatus は、未確定（0）の status code を 500 に補正します。
// error handler / recovery 後でも status が確定しないケースのフォールバックです。
func normalizeStatus(code int) int {
	if code == 0 {
		return http.StatusInternalServerError
	}
	return code
}

// statusClass は、status code を 1xx〜5xx の固定 enum に分類します。
// 範囲外の値は unknown に丸めます。
func statusClass(code int) string {
	switch {
	case code >= 100 && code < 200:
		return "1xx"
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return statusClassUnknown
	}
}

// statusCodeLabel は、status code を label 用の文字列に変換します。
// 0 以下は unknown に丸めます。
func statusCodeLabel(code int) string {
	if code <= 0 {
		return statusCodeUnknown
	}
	return strconv.Itoa(code)
}
