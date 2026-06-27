package uuid

import (
	"testing"

	"github.com/google/uuid"
)

// NewTestFromSalt はテスト専用の決定論UUID生成関数です。
//
// v5(SHA-1)ベースで、同じsaltなら毎回同じ値を返します。
func NewTestFromSalt(tb testing.TB, salt string) UUID {
	tb.Helper()
	ns := uuid.NameSpaceURL
	g := uuid.NewSHA1(ns, []byte(salt))
	return fromGoogle(g)
}
