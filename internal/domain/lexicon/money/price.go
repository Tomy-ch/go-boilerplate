// Package money は、金額の値オブジェクトを提供します。単価は価格スケール（サブセント可の Decimal）で保持し、
// 決済スケール（通貨の最小単位整数）への変換を担います。非負という金額の意味論は本層が持ち、
// 最小単位の桁数（USD=2 / JPY=0）は呼び出し元が policy として渡します。器としての正確な十進量は pkg/decimal に委ねます。
package money

import (
	"go-boilerplate/pkg/decimal"
)

// Price は、非負の単価を表す値オブジェクトです。サブセント精度（1 最小単位未満の単価）を保持できます。
type Price struct {
	amount decimal.Decimal
}

// NewPrice は、非負の Decimal から Price を構築します。負値の場合は ErrNegativePrice を返します。
func NewPrice(amount decimal.Decimal) (Price, error) {
	if amount.IsNegative() {
		return Price{}, ErrNegativePrice
	}
	return Price{amount: amount}, nil
}

// Decimal は、単価の十進量を返します。
func (p Price) Decimal() decimal.Decimal { return p.amount }

// String は、単価を十進文字列へ変換します。
func (p Price) String() string { return p.amount.String() }

// ToMinorUnit は、単価を通貨の最小単位（小数 n 桁）へ 0 から遠い方向へ丸めた決済スケールの整数へ変換します。
// n は 0 以上を前提とし、負値は ErrInvalidMinorUnit を返します。結果が int64 の範囲外になる場合はエラーを返します。
func (p Price) ToMinorUnit(n int32) (int64, error) {
	if n < 0 {
		return 0, ErrInvalidMinorUnit
	}
	return p.amount.ToScaledInt64(n)
}
