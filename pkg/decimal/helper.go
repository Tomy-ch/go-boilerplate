package decimal

import "github.com/shopspring/decimal"

func fromShopspring(d decimal.Decimal) Decimal {
	return Decimal{d: d}
}
