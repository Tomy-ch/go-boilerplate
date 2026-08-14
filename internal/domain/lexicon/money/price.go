// Package money は、金額の値オブジェクトを提供します。単価は価格スケール（サブセント可の Decimal）で保持し、
// 決済スケール（通貨の最小単位整数）への変換を担います。非負であることと決済スケールへ落とせることという
// 金額の意味論は本層が持ち、どの桁数へ落とすかは呼び出し元が policy として渡します。
// 器としての正確な十進量は pkg/decimal に委ねます。
package money

import (
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/xerrors"
)

// maxMinorUnitDigits は、この system が扱う通貨のうち最小単位が最も細かいものの小数桁数です。
// 桁数が大きいほど決済スケールの整数は大きくなるため、この値で変換できる金額はどの通貨でも変換できます。
const maxMinorUnitDigits = 2

// Price は、非負かつ決済スケールへ落とせる単価を表す値オブジェクトです。
// サブセント精度（1 最小単位未満の単価）を保持できます。
type Price struct {
	amount decimal.Decimal
}

// NewPrice は、非負の Decimal から Price を構築します。
// 負値の場合は ErrNegativePrice を、決済スケールの整数へ落とせない大きさの場合は ErrPriceOutOfRange を返します。
//
// 大きさをここで拒むのは、決済スケールへ落とせない単価が金額として成立しないためです。構築を許すと、
// 変換を試みる時点まで不正が持ち越され、その時点にいる呼び出し元（取得や集計）は拒否する術を持ちません。
func NewPrice(amount decimal.Decimal) (Price, error) {
	if amount.IsNegative() {
		return Price{}, ErrNegativePrice
	}
	if _, err := amount.ToScaledInt64(maxMinorUnitDigits); err != nil {
		return Price{}, xerrors.Wrap(ErrPriceOutOfRange, err.Error())
	}
	return Price{amount: amount}, nil
}

// Decimal は、単価の十進量を返します。
func (p Price) Decimal() decimal.Decimal { return p.amount }

// String は、単価を十進文字列へ変換します。
func (p Price) String() string { return p.amount.String() }

// ToMinorUnit は、単価を通貨の最小単位（小数 n 桁）へ 0 から遠い方向へ丸めた決済スケールの整数へ変換します。
// n は 0 以上を前提とし、負値は ErrInvalidMinorUnit を返します。
//
// 単価 1 件の変換は構築時の検証により成功しますが、n が maxMinorUnitDigits を超える場合は範囲外になり得ます。
func (p Price) ToMinorUnit(n int32) (int64, error) {
	if n < 0 {
		return 0, ErrInvalidMinorUnit
	}
	return p.amount.ToScaledInt64(n)
}
