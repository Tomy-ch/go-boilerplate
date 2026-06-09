package datetime

import (
	"errors"
	"time"
)

// inLocation は、parse の結果を loc で表示し直した time.Time を返す共通ヘルパです。
// loc が nil の場合、time.Time.In の panic を避けてエラーを返します
// （本関数群は error を返す契約のため、nil 入力も panic ではなくエラーで表面化させる）。
func inLocation(loc *time.Location, parse func() (time.Time, error)) (time.Time, error) {
	if loc == nil {
		return time.Time{}, errors.New("datetime: loc must not be nil")
	}

	t, err := parse()
	if err != nil {
		return time.Time{}, err
	}

	return t.In(loc), nil
}

// ParseRFC3339InLocation は、指定されたロケーションでRFC3339形式の日時文字列を解析します。
func ParseRFC3339InLocation(s string, loc *time.Location) (time.Time, error) {
	return inLocation(loc, func() (time.Time, error) { return ParseRFC3339(s) })
}

// ParseRFC3339UTCInLocation は、指定されたロケーションでRFC3339UTC形式の日時文字列を解析します。
func ParseRFC3339UTCInLocation(s string, loc *time.Location) (time.Time, error) {
	return inLocation(loc, func() (time.Time, error) { return ParseRFC3339UTC(s) })
}

// ParseRFC3339NanoInLocation は、指定されたロケーションでRFC3339Nano形式の日時文字列を解析します。
func ParseRFC3339NanoInLocation(s string, loc *time.Location) (time.Time, error) {
	return inLocation(loc, func() (time.Time, error) { return ParseRFC3339Nano(s) })
}

// ParseISO8601InLocation は、指定されたロケーションでISO8601形式の日時文字列を解析します。
func ParseISO8601InLocation(s string, loc *time.Location) (time.Time, error) {
	return inLocation(loc, func() (time.Time, error) { return ParseISO8601(s) })
}

// ParseDateTimeInLocation は、指定されたロケーションで日付と時刻の日時文字列を解析します。
func ParseDateTimeInLocation(s string, loc *time.Location) (time.Time, error) {
	return inLocation(loc, func() (time.Time, error) { return ParseDateTime(s) })
}

// ParseDateOnlyInLocation は、指定されたロケーションで日付のみの日時文字列を解析します。
func ParseDateOnlyInLocation(s string, loc *time.Location) (time.Time, error) {
	return inLocation(loc, func() (time.Time, error) { return ParseDateOnly(s) })
}

// ParseCustomLayoutInLocation は、指定されたロケーションでカスタムレイアウトの日時文字列を解析します。
func ParseCustomLayoutInLocation(layout, s string, loc *time.Location) (time.Time, error) {
	return inLocation(loc, func() (time.Time, error) { return ParseCustomLayout(layout, s) })
}
