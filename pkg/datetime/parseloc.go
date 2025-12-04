package datetime

import "time"

// ParseRFC3339InLocation は、指定されたロケーションでRFC3339形式の日時文字列を解析します。
func ParseRFC3339InLocation(s string, loc *time.Location) (time.Time, error) {
	t, err := ParseRFC3339(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// ParseRFC3339UTCInLocation は、指定されたロケーションでRFC3339UTC形式の日時文字列を解析します。
func ParseRFC3339UTCInLocation(s string, loc *time.Location) (time.Time, error) {
	t, err := ParseRFC3339UTC(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// ParseRFC3339NanoInLocation は、指定されたロケーションでRFC3339Nano形式の日時文字列を解析します。
func ParseRFC3339NanoInLocation(s string, loc *time.Location) (time.Time, error) {
	t, err := ParseRFC3339Nano(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// ParseISO8601InLocation は、指定されたロケーションでISO8601形式の日時文字列を解析します。
func ParseISO8601InLocation(s string, loc *time.Location) (time.Time, error) {
	t, err := ParseISO8601(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// ParseDateTimeInLocation は、指定されたロケーションで日付と時刻の日時文字列を解析します。
func ParseDateTimeInLocation(s string, loc *time.Location) (time.Time, error) {
	t, err := ParseDateTime(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// ParseDateOnlyInLocation は、指定されたロケーションで日付のみの日時文字列を解析します。
func ParseDateOnlyInLocation(s string, loc *time.Location) (time.Time, error) {
	t, err := ParseDateOnly(s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}

// ParseCustomLayoutInLocation は、指定されたロケーションでカスタムレイアウトの日時文字列を解析します。
func ParseCustomLayoutInLocation(layout, s string, loc *time.Location) (time.Time, error) {
	t, err := ParseCustomLayout(layout, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(loc), nil
}
