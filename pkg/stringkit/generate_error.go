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

// ValidateInRange は 文字数が minLen～maxLen 内かを判定し、範囲外なら理由メッセージを返します。
func ValidateInRange(s string, minLen, maxLen int) (bool, string) {
	n := RuneCount(s)
	if minLen <= n && n <= maxLen {
		return true, ""
	}
	return false, fmt.Sprintf(errMsgInRange, minLen, maxLen, n)
}

// ErrorMsgInRange は、文字数が minLen ～ maxLen の範囲外であった場合に返すエラーメッセージを生成します。
func ErrorMsgInRange(minLen, maxLen int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgInRange, minLen, maxLen, n)
}

// ErrorMsgMaxOrLess は、文字数が maxLen を超えた場合に返すエラーメッセージを生成します。
func ErrorMsgMaxOrLess(maxLen int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgMaxOrLess, maxLen, n)
}

// ErrorMsgMinOrMore は、文字数が minLen 未満であった場合に返すエラーメッセージを生成します。
func ErrorMsgMinOrMore(minLen int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgMinOrMore, minLen, n)
}

// ErrorMsgStrictInRange は、文字数が minLen < 文字数 < maxLen の範囲に収まらなかった場合に返すエラーメッセージを生成します。
func ErrorMsgStrictInRange(minLen, maxLen int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgStrictInRange, minLen, maxLen, n)
}

// ErrorMsgLessThanMax は、文字数が maxLen 以上だった場合に返すエラーメッセージを生成します。
func ErrorMsgLessThanMax(maxLen int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgLessThanMax, maxLen, n)
}

// ErrorMsgGreaterThanMin は、文字数が minLen 以下だった場合に返すエラーメッセージを生成します。
func ErrorMsgGreaterThanMin(minLen int, got string) string {
	n := RuneCount(got)
	return fmt.Sprintf(errMsgGreaterThanMin, minLen, n)
}
