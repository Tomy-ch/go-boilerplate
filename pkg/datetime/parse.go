// Package datetime は、RFC3339、ISO 8601、YYYY-MM-DD HH:MM:SS などの形式の日時文字列を解析するユーティリティを提供します。タイムゾーン変換付きの ParseXxxToLocation 系関数も含みます。
package datetime

import (
	"time"
)

// ParseRFC3339 は、RFC3339形式の日時文字列を解析します。
func ParseRFC3339(s string) (time.Time, error) {
	return ParseCustomLayout(time.RFC3339, s)
}

// ParseRFC3339UTC は、末尾が Z（UTC 表記）の RFC3339 文字列のみを解析します。
// オフセット付き入力（例 +09:00）は受理しません。
func ParseRFC3339UTC(s string) (time.Time, error) {
	return ParseCustomLayout("2006-01-02T15:04:05Z", s)
}

// ParseRFC3339Nano は、RFC3339Nano形式の日時文字列を解析します。
func ParseRFC3339Nano(s string) (time.Time, error) {
	return ParseCustomLayout(time.RFC3339Nano, s)
}

// ParseISO8601 は、ISO8601形式の日時文字列を解析します。
// オフセットは Z または ±hhmm 形式（コロンなし）のみ受理します。±hh:mm 形式（例 +09:00）は解析エラーになります。
func ParseISO8601(s string) (time.Time, error) {
	return ParseCustomLayout("2006-01-02T15:04:05Z0700", s)
}

// ParseDateTime は、日付と時刻（YYYY-MM-DD HH:MM:SS）の日時文字列を解析します。
func ParseDateTime(s string) (time.Time, error) {
	return ParseCustomLayout(time.DateTime, s)
}

// ParseDateOnly は、日付のみ（YYYY-MM-DD）の日時文字列を解析します。
func ParseDateOnly(s string) (time.Time, error) {
	return ParseCustomLayout(time.DateOnly, s)
}

// ParseCustomLayout は、カスタムレイアウトの日時文字列を解析します。
// layout は Go の参照時刻（2006-01-02T15:04:05Z07:00）に基づくフォーマット文字列です（time.Parse と同仕様）。
func ParseCustomLayout(layout, s string) (time.Time, error) {
	return time.Parse(layout, s)
}
