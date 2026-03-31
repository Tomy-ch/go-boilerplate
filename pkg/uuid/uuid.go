// Package uuid は、UUIDを生成および操作するための機能を提供します。
package uuid

import (
	"database/sql/driver"
	"testing"

	"github.com/google/uuid"
)

//nolint:recvcheck // safe: UUID is immutable (value object); pointer receiver is required only for Scan to implement sql.Scanner
type UUID struct{ b [16]byte }

// New は、uuidを生成します。生成に失敗した場合はエラーを返します。
func New() (UUID, error) {
	g, err := uuid.NewV7()
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
	ns := uuid.NameSpaceURL
	g := uuid.NewSHA1(ns, []byte(salt))
	return fromGoogle(g)
}

// String は、UUIDを文字列に変換します。
func (u UUID) String() string { return toGoogle(u).String() }

// Bytes は、UUIDをバイト配列に変換します。
func (u UUID) Bytes() [16]byte { return u.b }

// ToPrimitive は、UUIDから github.com/google/uuid の uuid.UUID を生成します。
func (u UUID) ToPrimitive() uuid.UUID { return toGoogle(u) }

// IsNil は、UUIDが全てゼロであるかどうかを判定します。
func (u UUID) IsNil() bool { return u.b == [16]byte{} }

// Equal は、引数のUUIDと等しいかどうかを判定します。
func (u UUID) Equal(v UUID) bool { return u.b == v.b }

// ToPtr は、UUIDのポインタを返します。
func (u UUID) ToPtr() *UUID { return &u }

// EqualPtr は、ポインタを介してUUIDが等しいかどうかを判定します。
func (u UUID) EqualPtr(v *UUID) bool { return v != nil && u.b == v.b }

// Parse は、文字列からUUIDを解析します。解析に失敗した場合はエラーを返します。
func Parse(s string) (UUID, error) {
	g, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, err
	}
	return fromGoogle(g), nil
}

// Scan は、データベースからUUIDをスキャンします。スキャンに失敗した場合はエラーを返します。
func (u *UUID) Scan(src any) error {
	var g uuid.UUID
	if err := g.Scan(src); err != nil {
		return err
	}
	u.b = g
	return nil
}

// Value は、UUIDをデータベースに保存するための値に変換します。変換に失敗した場合はエラーを返します。
func (u UUID) Value() (driver.Value, error) {
	return toGoogle(u).Value()
}
