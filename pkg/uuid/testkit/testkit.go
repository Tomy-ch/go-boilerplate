// Package testkit は、uuid のテスト補助を提供します。production から import されないため、
// testing パッケージが production バイナリへリンクされません（pkg/uuid 本体からの隔離）。
package testkit

import (
	"crypto/sha256"
	"testing"

	guuid "github.com/google/uuid"

	"go-boilerplate/pkg/uuid"
)

// NewTestFromSalt はテスト専用の決定論UUID生成関数です。
//
// SHA-256 ベースの UUID v8 を返すため、同じ salt では毎回同じ値になります。
func NewTestFromSalt(tb testing.TB, salt string) uuid.UUID {
	tb.Helper()
	return uuid.FromPrimitive(guuid.NewHash(sha256.New(), guuid.NameSpaceURL, []byte(salt), 8))
}
