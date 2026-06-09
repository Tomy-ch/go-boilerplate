package datetime

import (
	"errors"
	"time"
)

// toLocation は、parse で得た時刻を loc のタイムゾーンへ変換して返す共通ヘルパです。
// loc が nil の場合、time.Time.In の panic を避けてエラーを返します。
func toLocation(loc *time.Location, parse func() (time.Time, error)) (time.Time, error) {
	if loc == nil {
		return time.Time{}, errors.New("datetime: loc must not be nil")
	}

	t, err := parse()
	if err != nil {
		return time.Time{}, err
	}

	return t.In(loc), nil
}

// ParseRFC3339ToLocation は、RFC3339形式の日時文字列を解析し loc のタイムゾーンへ変換します。
func ParseRFC3339ToLocation(s string, loc *time.Location) (time.Time, error) {
	return toLocation(loc, func() (time.Time, error) { return ParseRFC3339(s) })
}

// ParseRFC3339UTCToLocation は、RFC3339UTC形式の日時文字列を解析し loc のタイムゾーンへ変換します。
func ParseRFC3339UTCToLocation(s string, loc *time.Location) (time.Time, error) {
	return toLocation(loc, func() (time.Time, error) { return ParseRFC3339UTC(s) })
}

// ParseRFC3339NanoToLocation は、RFC3339Nano形式の日時文字列を解析し loc のタイムゾーンへ変換します。
func ParseRFC3339NanoToLocation(s string, loc *time.Location) (time.Time, error) {
	return toLocation(loc, func() (time.Time, error) { return ParseRFC3339Nano(s) })
}

// ParseISO8601ToLocation は、ISO8601形式の日時文字列を解析し loc のタイムゾーンへ変換します。
func ParseISO8601ToLocation(s string, loc *time.Location) (time.Time, error) {
	return toLocation(loc, func() (time.Time, error) { return ParseISO8601(s) })
}

// ParseDateTimeToLocation は、日付と時刻の日時文字列を解析し loc のタイムゾーンへ変換します。
func ParseDateTimeToLocation(s string, loc *time.Location) (time.Time, error) {
	return toLocation(loc, func() (time.Time, error) { return ParseDateTime(s) })
}

// ParseDateOnlyToLocation は、日付のみの日時文字列を解析し loc のタイムゾーンへ変換します。
func ParseDateOnlyToLocation(s string, loc *time.Location) (time.Time, error) {
	return toLocation(loc, func() (time.Time, error) { return ParseDateOnly(s) })
}

// ParseCustomLayoutToLocation は、カスタムレイアウトの日時文字列を解析し loc のタイムゾーンへ変換します。
func ParseCustomLayoutToLocation(layout, s string, loc *time.Location) (time.Time, error) {
	return toLocation(loc, func() (time.Time, error) { return ParseCustomLayout(layout, s) })
}
