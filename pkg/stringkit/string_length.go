// Package stringkit は、文字列の長さを検証するための関数を提供します。
package stringkit

import (
	"unicode/utf8"
)

// RuneCount は UTF-8 文字列の文字数を返します。
func RuneCount(s string) int {
	return utf8.RuneCountInString(s)
}

// InRange は minLen <= 文字数 <= maxLen のとき true
func InRange(s string, minLen, maxLen int) bool {
	n := RuneCount(s)
	return minLen <= n && n <= maxLen
}

// MaxOrLess は 文字数 <= maxLen のとき true
func MaxOrLess(s string, maxLen int) bool {
	return RuneCount(s) <= maxLen
}

// MinOrMore は 文字数 >= minLen のとき true
func MinOrMore(s string, minLen int) bool {
	return RuneCount(s) >= minLen
}

// StrictInRange は minLen < 文字数 < maxLen のとき true
func StrictInRange(s string, minLen, maxLen int) bool {
	n := RuneCount(s)
	return minLen < n && n < maxLen
}

// LessThanMax は 文字数 < maxLen のとき true
func LessThanMax(s string, maxLen int) bool {
	return RuneCount(s) < maxLen
}

// GreaterThanMin は 文字数 > minLen のとき true
func GreaterThanMin(s string, minLen int) bool {
	return RuneCount(s) > minLen
}
