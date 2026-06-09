// Package datetime は、日付と時刻に関連する機能を提供します。
package datetime

import (
	"time"
)

// ParseRFC3339 は、RFC3339形式の日時文字列を解析します。
func ParseRFC3339(s string) (time.Time, error) {
	return ParseCustomLayout(time.RFC3339, s)
}

// ParseRFC3339UTC は、末尾が Z（UTC 表記）の RFC3339 文字列のみを解析します。
// レイアウト末尾はリテラル Z 固定のため、オフセット付き入力（例 +09:00）は受理しません。
func ParseRFC3339UTC(s string) (time.Time, error) {
	return ParseCustomLayout("2006-01-02T15:04:05Z", s)
}

// ParseRFC3339Nano は、RFC3339Nano形式の日時文字列を解析します。
func ParseRFC3339Nano(s string) (time.Time, error) {
	return ParseCustomLayout(time.RFC3339Nano, s)
}

// ParseISO8601 は、ISO8601形式の日時文字列を解析します。
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
// 本パッケージの名前付きラッパ（ParseRFC3339 等）が共通で通す解析窓口であり、
// 将来の共通エラーラップ等の拡張点も兼ねる（現状は time.Parse の素通し）。
func ParseCustomLayout(layout, s string) (time.Time, error) {
	return time.Parse(layout, s)
}
