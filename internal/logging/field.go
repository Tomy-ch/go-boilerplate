package logging

import (
	"boilerplate-go/pkg/xerrors"
)

const (
	fieldUnknown fieldKind = iota
	fieldString
	fieldStrings
	fieldInt
	fieldInt64
	fieldFloat64
	fieldBool
	fieldError
	fieldAny
)

// fieldKind は、フィールドの種類を表します。
type fieldKind int

// Field は、ログフィールドを表します。
type Field struct {
	key  string
	kind fieldKind

	stringValue  string
	stringsValue []string
	intValue     int
	int64Value   int64
	float64Value float64
	boolValue    bool
	errorValue   error
	anyValue     any
}

// String は、文字列のログフィールドを作成します。
func String(key, v string) *Field {
	return &Field{
		key:         key,
		kind:        fieldString,
		stringValue: v,
	}
}

// Int は、整数のログフィールドを作成します。
func Int(key string, v int) *Field {
	return &Field{
		key:      key,
		kind:     fieldInt,
		intValue: v,
	}
}

// Strings は、文字列のスライスのログフィールドを作成します。
func Strings(key string, v []string) *Field {
	return &Field{
		key:          key,
		kind:         fieldStrings,
		stringsValue: v,
	}
}

// Int64 は、64ビット整数のログフィールドを作成します。
func Int64(key string, v int64) *Field {
	return &Field{
		key:        key,
		kind:       fieldInt64,
		int64Value: v,
	}
}

// Float64 は、64ビット浮動小数点数のログフィールドを作成します。
func Float64(key string, v float64) *Field {
	return &Field{
		key:          key,
		kind:         fieldFloat64,
		float64Value: v,
	}
}

// Bool は、真偽値のログフィールドを作成します。
func Bool(key string, v bool) *Field {
	return &Field{
		key:       key,
		kind:      fieldBool,
		boolValue: v,
	}
}

// Error は、エラーのログフィールドを作成します。
func Error(key string, err error) *Field {
	return &Field{
		key:        key,
		kind:       fieldError,
		errorValue: err,
	}
}

// Stacktrace は、エラースタックトレースのログフィールドを作成します。
func Stacktrace(key string, err error) *Field {
	return &Field{
		key:         key,
		kind:        fieldString,
		stringValue: xerrors.StackTrace(err),
	}
}

// Any は、任意の型のログフィールドを作成します。
func Any(key string, v any) *Field {
	return &Field{
		key:      key,
		kind:     fieldAny,
		anyValue: v,
	}
}
