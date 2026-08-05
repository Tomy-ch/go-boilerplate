// Package testkit は、uuid のテスト補助を提供します。production から import されないため、
// testing パッケージが production バイナリへリンクされません（pkg/uuid 本体からの隔離）。
package testkit

import (
	"testing"

	guuid "github.com/google/uuid"

	"go-boilerplate/pkg/uuid"
)

// NewTestFromSalt はテスト専用の決定論UUID生成関数です。
//
// v5(SHA-1)ベースで、同じsaltなら毎回同じ値を返します。
func NewTestFromSalt(tb testing.TB, salt string) uuid.UUID {
	tb.Helper()
	return uuid.FromPrimitive(guuid.NewSHA1(guuid.NameSpaceURL, []byte(salt)))
}
