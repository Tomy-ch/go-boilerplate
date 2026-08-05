package purchase

const (
	// taxRatePercent は、国内消費税率（パーセント）です。sample の placeholder であり、
	// 要件化した時点で config / マスタへ移します（docs/spec/purchase/domain.md）。
	taxRatePercent = 10

	// shippingFeeCents は、固定送料（USD セント）です。sample の placeholder です
	// （docs/spec/purchase/domain.md）。
	shippingFeeCents = 500

	// percentDivisor は、パーセント計算の除数です（taxRatePercent を百分率として扱うため）。
	percentDivisor = 100

	// minorUnitDigits は、決済通貨（USD）の最小単位の小数桁数です（セント = 小数 2 桁）。
	// 価格スケール（ドル decimal）から決済スケール（整数セント）へ切り捨てる際の桁数に用います。
	minorUnitDigits = 2

	// minQuantity は、明細 1 件あたりの最小購入数量です。
	minQuantity = 1
)
