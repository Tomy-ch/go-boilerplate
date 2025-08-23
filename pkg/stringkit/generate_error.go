package stringkit

import "fmt"

const (
	errMsgInRange        = "length must be between %d and %d characters (got %d)"
	errMsgMaxOrLess      = "length must be less than or equal to %d characters (got %d)"
	errMsgMinOrMore      = "length must be greater than or equal to %d characters (got %d)"
	errMsgStrictInRange  = "length must be greater than %d and less than %d characters (got %d)"
	errMsgLessThanMax    = "length must be less than %d characters (got %d)"
	errMsgGreaterThanMin = "length must be greater than %d characters (got %d)"
)

// ErrorMsgInRange は、文字数が lowerBound ～ upperBound の範囲外であった場合に返すエラーメッセージを生成します。
func ErrorMsgInRange(lowerBound, upperBound int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgInRange, lowerBound, upperBound, n)
}

// ErrorMsgMaxOrLess は、文字数が upperBound を超えた場合に返すエラーメッセージを生成します。
func ErrorMsgMaxOrLess(upperBound int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgMaxOrLess, upperBound, n)
}

// ErrorMsgMinOrMore は、文字数が lowerBound 未満であった場合に返すエラーメッセージを生成します。
func ErrorMsgMinOrMore(lowerBound int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgMinOrMore, lowerBound, n)
}

// ErrorMsgStrictInRange は、文字数が lowerBound < len < upperBound の範囲に収まらなかった場合に返すエラーメッセージを生成します。
func ErrorMsgStrictInRange(lowerBound, upperBound int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgStrictInRange, lowerBound, upperBound, n)
}

// ErrorMsgLessThanMax は、文字数が upperBound 以上だった場合に返すエラーメッセージを生成します。
func ErrorMsgLessThanMax(upperBound int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgLessThanMax, upperBound, n)
}

// ErrorMsgGreaterThanMin は、文字数が lowerBound 以下だった場合に返すエラーメッセージを生成します。
func ErrorMsgGreaterThanMin(lowerBound int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgGreaterThanMin, lowerBound, n)
}
