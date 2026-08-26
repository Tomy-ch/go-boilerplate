// Package uuid は、UUIDを生成および操作するための機能を提供します。
package uuid

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"

	"github.com/google/uuid"
)

// UUID は、128 ビットの一意識別子を表す値オブジェクトです。
type UUID struct{ b [16]byte } //nolint:recvcheck // immutable VO; pointer receiver only for Scan/UnmarshalJSON

// New は、UUIDv7（時刻単調増加）を生成します。生成に失敗した場合はエラーを返します。
func New() (UUID, error) {
	g, err := uuid.NewV7()
	if err != nil {
		return UUID{}, err
	}
	return fromGoogle(g), nil
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

// FromPrimitive は、github.com/google/uuid の uuid.UUID からドメインの UUID を生成します。
func FromPrimitive(g uuid.UUID) UUID { return fromGoogle(g) }

// Parse は、文字列からUUIDを解析します。解析に失敗した場合はエラーを返します。
func Parse(s string) (UUID, error) {
	g, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, err
	}
	return fromGoogle(g), nil
}

// MarshalJSON は、UUIDを正準文字列表現の JSON 文字列（例 "0190a1b2-..."）へ符号化します。
func (u UUID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + toGoogle(u).String() + `"`), nil
}

// UnmarshalJSON は、JSON 文字列からUUIDを復元します。復元に失敗した場合はエラーを返します。
// JSON null は値を変更しません。
func (u *UUID) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	g, err := uuid.Parse(s)
	if err != nil {
		return err
	}
	*u = fromGoogle(g)
	return nil
}

// Scan は、データベースからUUIDをスキャンします。スキャンに失敗した場合はエラーを返します。
func (u *UUID) Scan(src any) error {
	var g uuid.UUID
	if err := g.Scan(src); err != nil {
		return err
	}
	*u = fromGoogle(g)
	return nil
}

// Value は、UUIDをデータベースに保存するための値に変換します。変換に失敗した場合はエラーを返します。
func (u UUID) Value() (driver.Value, error) {
	return toGoogle(u).Value()
}
