// Package datetime は、日付と時刻に関連する機能を提供します。
package datetime

import (
	"time"
)

// ParseRFC3339 は、RFC3339形式の日時文字列のレイアウトを定義します。
func ParseRFC3339(s string) (time.Time, error) {
	return ParseCustomLayout(time.RFC3339, s)
}

// ParseRFC3339UTC は、RFC3339形式の日時文字列のレイアウトを定義します（UTCタイムゾーン固定）。
func ParseRFC3339UTC(s string) (time.Time, error) {
	return ParseCustomLayout("2006-01-02T15:04:05Z", s)
}

// ParseRFC3339Nano は、RFC3339Nano形式の日時文字列のレイアウトを定義します。
func ParseRFC3339Nano(s string) (time.Time, error) {
	return ParseCustomLayout(time.RFC3339Nano, s)
}

// ParseISO8601 は、ISO8601形式の日時文字列のレイアウトを定義します。
func ParseISO8601(s string) (time.Time, error) {
	return ParseCustomLayout("2006-01-02T15:04:05Z0700", s)
}

// ParseDateTime は、日付と時刻の日時文字列のレイアウトを定義します。
func ParseDateTime(s string) (time.Time, error) {
	return ParseCustomLayout(time.DateTime, s)
}

// ParseDateOnly は、日付のみの日時文字列のレイアウトを定義します。
func ParseDateOnly(s string) (time.Time, error) {
	return ParseCustomLayout(time.DateOnly, s)
}

// ParseCustomLayout は、カスタムレイアウトの日時文字列を解析します。
func ParseCustomLayout(layout, s string) (time.Time, error) {
	return time.Parse(layout, s)
}
