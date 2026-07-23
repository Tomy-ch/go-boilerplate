// Package money は、Usecase 層のマネー計算ヘルパを提供します。
//
// 正確な十進の算術・丸め・最小単位整数への変換といった純機構は pkg/decimal が担い、本パッケージは
// 「どの最小単位桁へ・どの丸めモードで」決済スケールへ落とすかという policy の選択のみを残します。
package money

import "go-boilerplate/pkg/decimal"

// ApplyRateHalfUp は、金額 amount にレート rate を掛け、最小単位（小数 minorUnitDigits 桁）へ
// half-up（0 から遠い方向）で丸めた決済スケールの整数を返します。
//
// 丸めは決済境界のこの 1 点でのみ行い、途中は正確な十進のまま計算するため float 誤差は生じません。
// 結果が int64 の範囲外になる場合はエラーを返します。
func ApplyRateHalfUp(amount, rate decimal.Decimal, minorUnitDigits int32) (int64, error) {
	return amount.Mul(rate).ToScaledInt64(minorUnitDigits)
}
