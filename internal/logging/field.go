package logging

import (
	"strings"
	"time"

	"go-boilerplate/pkg/xerrors"
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

type fieldKey string

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
func String(key fieldKey, v string) *Field {
	return &Field{
		key:         string(key),
		kind:        fieldString,
		stringValue: v,
	}
}

// Int は、整数のログフィールドを作成します。
func Int(key fieldKey, v int) *Field {
	return &Field{
		key:      string(key),
		kind:     fieldInt,
		intValue: v,
	}
}

// Strings は、文字列のスライスのログフィールドを作成します。
func Strings(key fieldKey, v []string) *Field {
	return &Field{
		key:          string(key),
		kind:         fieldStrings,
		stringsValue: v,
	}
}

// Int64 は、64ビット整数のログフィールドを作成します。
func Int64(key fieldKey, v int64) *Field {
	return &Field{
		key:        string(key),
		kind:       fieldInt64,
		int64Value: v,
	}
}

// Float64 は、64ビット浮動小数点数のログフィールドを作成します。
func Float64(key fieldKey, v float64) *Field {
	return &Field{
		key:          string(key),
		kind:         fieldFloat64,
		float64Value: v,
	}
}

// Bool は、真偽値のログフィールドを作成します。
func Bool(key fieldKey, v bool) *Field {
	return &Field{
		key:       string(key),
		kind:      fieldBool,
		boolValue: v,
	}
}

// Time は、時間のログフィールドを作成します。
func Time(key fieldKey, v time.Time) *Field {
	return &Field{
		key:         string(key),
		kind:        fieldString,
		stringValue: v.Format(time.RFC3339Nano),
	}
}

// DurationMs は、時間間隔のログフィールドをミリ秒単位で作成します。
func DurationMs(key fieldKey, v time.Duration) *Field {
	return &Field{
		key:          string(key),
		kind:         fieldFloat64,
		float64Value: latencyMs(v),
	}
}

// Error は、エラーのログフィールドを作成します。
func Error(key fieldKey, err error) *Field {
	return &Field{
		key:        string(key),
		kind:       fieldError,
		errorValue: err,
	}
}

// Stacktrace は、エラースタックトレースのログフィールドを作成します。
// JSON ビューア（Grafana / Loki 等）で改行が可読に表示されるよう、
// 単一文字列ではなく行ごとに分割した []string を保持します。
func Stacktrace(key fieldKey, err error) *Field {
	return &Field{
		key:          string(key),
		kind:         fieldStrings,
		stringsValue: SplitStackLines(xerrors.StackTrace(err)),
	}
}

// SplitStackLines は、スタックトレース文字列を改行で分割し行配列に変換します。
// 末尾の空行は除去し、空文字列または改行のみの入力では nil を返します。
// 配列化により行境界が構造として表現されるため、`\t<file>:<line>` の先頭タブは除去します。
func SplitStackLines(s string) []string {
	trimmed := strings.TrimRight(s, "\n")
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, "\t")
	}
	return lines
}

// Any は、任意の型のログフィールドを作成します。
func Any(key fieldKey, v any) *Field {
	return &Field{
		key:      string(key),
		kind:     fieldAny,
		anyValue: v,
	}
}

// latencyMs は、latency をミリ秒単位の float64 に変換します。
func latencyMs(latency time.Duration) float64 {
	return float64(latency) / float64(time.Millisecond)
}
