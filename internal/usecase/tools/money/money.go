// Package money は、Usecase 層のマネー計算ヘルパを提供します。
//
// マネー値は最小単位の整数（例: USD セント / JPY 円）で保持し、float による
// 累積誤差を避けます。丸め方式はアプリ規約（ADR）で確定した half-up に統一します。
package money

import (
	"math"
	"math/big"
)

// rateFixedScale は、レート（比率）を整数演算へ載せるための固定小数スケール（10^6）です。
const rateFixedScale = 1_000_000

// ApplyRateHalfUp は、最小単位整数 amountMinor に rate を適用し、scale で除して
// half-up（0 から遠い方向への四捨五入）した整数を返します。
//
// scale は amountMinor を得る際に用いた固定小数スケールです（例: amount×100 で
// セント化したなら 100）。中間積は多倍長整数で計算するため int64 の乗算オーバーフローは
// 起きません。amountMinor と rate は任意符号を許容し、負値は 0 から遠い方向へ丸めます。
// scale は正の整数を前提とします（0 以下はゼロ除算で panic）。
func ApplyRateHalfUp(amountMinor int64, rate float64, scale int64) int64 {
	rateFixed := int64(math.Round(rate * rateFixedScale))
	denom := big.NewInt(scale * rateFixedScale)
	product := new(big.Int).Mul(big.NewInt(amountMinor), big.NewInt(rateFixed))

	// 符号に応じて denom/2 を加減してから 0 方向へ切り捨て、half-away-from-zero を実現する。
	half := new(big.Int).Rsh(denom, 1)
	if product.Sign() < 0 {
		half.Neg(half)
	}
	product.Add(product, half)
	product.Quo(product, denom)
	return product.Int64()
}
