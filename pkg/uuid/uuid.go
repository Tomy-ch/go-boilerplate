// Package uuid は、UUIDを生成および操作するための機能を提供します。
package uuid

import (
	"testing"

	guuid "github.com/google/uuid"
)

type UUID struct{ b [16]byte }

// New は、uuidを生成します。生成に失敗した場合はエラーを返します。
func New() (UUID, error) {
	g, err := guuid.NewV7()
	if err != nil {
		return UUID{}, err
	}
	return fromGoogle(g), nil
}

// NewTestFromSalt はテスト専用の決定論UUID生成関数です。
//
// 本番利用は想定していません。
// v5(SHA-1)ベースで、同じsaltなら毎回同じ値を返します。
func NewTestFromSalt(t testing.TB, salt string) UUID {
	t.Helper()
	ns := guuid.NameSpaceURL
	g := guuid.NewSHA1(ns, []byte(salt))
	return fromGoogle(g)
}

// String は、UUIDを文字列に変換します。
func (u UUID) String() string { return toGoogle(u).String() }

// Equal は、引数のUUIDと等しいかどうかを判定します。
func (u UUID) Equal(v UUID) bool { return u.b == v.b }

// ToPtr は、UUIDのポインタを返します。
func (u UUID) ToPtr() *UUID { return &u }

// EqualPtr は、ポインタを介してUUIDが等しいかどうかを判定します。
func (u UUID) EqualPtr(v *UUID) bool { return v != nil && u.b == v.b }

// Parse は、文字列からUUIDを解析します。解析に失敗した場合はエラーを返します。
func Parse(s string) (UUID, error) {
	g, err := guuid.Parse(s)
	if err != nil {
		return UUID{}, err
	}
	return fromGoogle(g), nil
}
