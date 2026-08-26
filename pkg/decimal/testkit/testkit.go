// Package testkit は、decimal のテスト補助を提供します。production から import されないため、
// testing パッケージが production バイナリへリンクされません（pkg/decimal 本体からの隔離）。
package testkit

import (
	"testing"

	"go-boilerplate/pkg/decimal"
)

// MustParse はテスト専用の Decimal 生成関数です。s の解析に失敗した場合はテストを即座に失敗させます。
func MustParse(tb testing.TB, s string) decimal.Decimal {
	tb.Helper()
	d, err := decimal.Parse(s)
	if err != nil {
		tb.Fatalf("decimal.testkit.MustParse(%q): %v", s, err)
	}
	return d
}
