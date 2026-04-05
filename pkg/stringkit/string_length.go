// Package stringkit は、文字列の長さを検証するための関数を提供します。
package stringkit

import (
	"unicode/utf8"
)

// RuneCount は UTF-8 文字列の文字数を返します。
func RuneCount(s string) int {
	return utf8.RuneCountInString(s)
}

// InRange は minLBound <= len <= maxLBound のとき true
func InRange(s string, minLBound, maxLBound int) bool {
	n := RuneCount(s)
	return minLBound <= n && n <= maxLBound
}

// MaxOrLess は len <= maxLBound のとき true
func MaxOrLess(s string, maxLBound int) bool {
	return RuneCount(s) <= maxLBound
}

// MinOrMore は len >= minLBound のとき true
func MinOrMore(s string, minLBound int) bool {
	return RuneCount(s) >= minLBound
}

// StrictInRange は minLBound < len < maxLBound のとき true
func StrictInRange(s string, minLBound, maxLBound int) bool {
	n := RuneCount(s)
	return minLBound < n && n < maxLBound
}

// LessThanMax は len < maxLBound のとき true
func LessThanMax(s string, maxLBound int) bool {
	return RuneCount(s) < maxLBound
}

// GreaterThanMin は len > minLBound のとき true
func GreaterThanMin(s string, minLBound int) bool {
	return RuneCount(s) > minLBound
}
