// Package decimal は、金額・比率などの正確な十進量を扱う exact-decimal 値オブジェクトを提供します。
//
// shopspring/decimal をラップして vendor 依存を隠蔽する seam であり、application コードは
// 本パッケージ経由でのみ十進量を扱います。通貨・非負・最小単位といった業務意味論は持たず、
// 純粋な十進算術（算術・丸め・スケール変換・境界往復）だけを担います。ゼロ値は 0 を表します。
package decimal

import (
	"database/sql/driver"

	"go-boilerplate/pkg/xerrors"

	"github.com/shopspring/decimal"
)

// 受理する十進値の桁数上限です。PostgreSQL の NUMERIC が保持できる桁数（小数点前 131072 桁 /
// 小数点後 16383 桁）に合わせています。
//
// 上限が要るのは、指数表記が「解析は一瞬・文字列化は爆発」という非対称性を持つためです。
// 例えば "1E100000000" は係数と指数だけを保持するため即座に解析できますが、String() は
// 約 1 億桁を materialize しようとしてプロセスを停止させます。
const (
	maxIntegerDigits  = 131072
	maxFractionDigits = 16383
)

// Decimal は、正確な十進量を表す不変の値オブジェクトです。ゼロ値は 0 を表します。
type Decimal struct{ d decimal.Decimal } //nolint:recvcheck // immutable VO; pointer receiver only for Scan/UnmarshalJSON

// Parse は、十進文字列を Decimal へ解析します。解析に失敗した場合はエラーを返します。
//
//	桁数が永続化可能な範囲を超える値も ErrInvalid として拒否します。
func Parse(s string) (Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Decimal{}, xerrors.Wrap(ErrInvalid, err.Error())
	}
	if err := checkMagnitude(d); err != nil {
		return Decimal{}, err
	}
	return fromShopspring(d), nil
}

// checkMagnitude は、値の桁数が受理範囲に収まるかを検証します。
func checkMagnitude(d decimal.Decimal) error {
	exp := int(d.Exponent())
	if exp < 0 {
		if -exp > maxFractionDigits {
			return xerrors.Wrap(ErrInvalid, "too many fraction digits")
		}
		return nil
	}
	if d.NumDigits()+exp > maxIntegerDigits {
		return xerrors.Wrap(ErrInvalid, "too many integer digits")
	}
	return nil
}

// FromInt は、int64 から Decimal を生成します。
func FromInt(i int64) Decimal { return fromShopspring(decimal.NewFromInt(i)) }

// String は、Decimal を十進文字列へ変換します。
func (d Decimal) String() string { return d.d.String() }

// Add は、d と o の和を返します。
func (d Decimal) Add(o Decimal) Decimal { return fromShopspring(d.d.Add(o.d)) }

// Sub は、d から o を引いた差を返します。
func (d Decimal) Sub(o Decimal) Decimal { return fromShopspring(d.d.Sub(o.d)) }

// Mul は、d と o の積を返します。
func (d Decimal) Mul(o Decimal) Decimal { return fromShopspring(d.d.Mul(o.d)) }

// Neg は、d の符号を反転した値を返します。
func (d Decimal) Neg() Decimal { return fromShopspring(d.d.Neg()) }

// DivRound は、d を o で除算し places 桁で 0 から遠い方向へ丸めた商を返します。o が 0 の場合は panic します。
func (d Decimal) DivRound(o Decimal, places int32) Decimal {
	return fromShopspring(d.d.DivRound(o.d, places))
}

// RoundHalfAwayFromZero は、places 桁で 0 から遠い方向へ四捨五入した値を返します。
func (d Decimal) RoundHalfAwayFromZero(places int32) Decimal {
	return fromShopspring(d.d.Round(places))
}

// Truncate は、places 桁で 0 方向へ切り捨てた値を返します。
func (d Decimal) Truncate(places int32) Decimal { return fromShopspring(d.d.Truncate(places)) }

// Cmp は、d と o を比較し、d<o なら -1、d==o なら 0、d>o なら 1 を返します。
func (d Decimal) Cmp(o Decimal) int { return d.d.Cmp(o.d) }

// Equal は、d と o が数値として等しいかどうかを判定します。
func (d Decimal) Equal(o Decimal) bool { return d.d.Equal(o.d) }

// Sign は、d の符号を返します（負なら -1、0 なら 0、正なら 1）。
func (d Decimal) Sign() int { return d.d.Sign() }

// IsZero は、d が 0 であるかどうかを判定します。
func (d Decimal) IsZero() bool { return d.d.IsZero() }

// IsNegative は、d が負であるかどうかを判定します。
func (d Decimal) IsNegative() bool { return d.d.Sign() < 0 }

// ToScaledInt64 は、n 桁で 0 から遠い方向へ丸めたうえで 10^n を掛け、最小単位整数へ変換します。
// n は 0 以上を前提とします。結果が int64 の範囲外になる場合は ErrOverflow を返します。
func (d Decimal) ToScaledInt64(n int32) (int64, error) {
	scaled := d.d.Round(n).Shift(n)
	bi := scaled.BigInt()
	if !bi.IsInt64() {
		return 0, xerrors.Wrap(ErrOverflow, "scaled value "+scaled.String()+" exceeds int64 range")
	}
	return bi.Int64(), nil
}

// MarshalJSON は、Decimal を JSON 文字列（例 "19.99"）へ符号化します。
// JSON number は IEEE754 double として復元され値が壊れるため、常に文字列で表現します。
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.d.String() + `"`), nil
}

// UnmarshalJSON は、JSON 文字列または JSON number から Decimal を復元します。
// number も桁を保持したまま取り込むため、可逆性を損ないません。
func (d *Decimal) UnmarshalJSON(b []byte) error {
	var inner decimal.Decimal
	if err := inner.UnmarshalJSON(b); err != nil {
		return xerrors.Wrap(ErrInvalid, err.Error())
	}
	d.d = inner
	return nil
}

// Scan は、データベースの NUMERIC 値を Decimal へ読み込みます。読み込みに失敗した場合はエラーを返します。
func (d *Decimal) Scan(src any) error {
	var inner decimal.Decimal
	if err := inner.Scan(src); err != nil {
		return xerrors.Wrap(ErrInvalid, err.Error())
	}
	d.d = inner
	return nil
}

// Value は、Decimal をデータベースの NUMERIC へ保存する値へ変換します。
func (d Decimal) Value() (driver.Value, error) {
	return d.d.Value()
}
