package cart

import (
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/xerrors"
)

// PurchasableLine は、小計に入る明細 1 件の観測値です。
type PurchasableLine struct {
	quantity int
	price    money.Price
}

// NewPurchasableLine は、小計に入る明細の観測値を組み立てます。
func NewPurchasableLine(quantity int, price money.Price) PurchasableLine {
	return PurchasableLine{quantity: quantity, price: price}
}

// Subtotal は、購入可能な明細を合算して決済スケールの整数へ落とします。
// 丸めは合算の後に一度だけ行い、明細ごとの丸め誤差が積み上がらないようにします。
//
// 幅を超えるのは明細が積み上がった結果に限られます（単価 1 件が決済スケールへ落とせることは
// money.Price の構築時に保証されるため）。合計を偽らずに返す術が無いため、エラーを返します。
func Subtotal(lines []PurchasableLine) (int64, error) {
	sum := decimal.FromInt(0)
	for _, l := range lines {
		sum = sum.Add(l.price.Decimal().Mul(decimal.FromInt(int64(l.quantity))))
	}

	subtotal, err := sum.ToScaledInt64(subtotalMinorUnitDigits)
	if err != nil {
		return 0, xerrors.Join(ErrSubtotalOutOfRange, err)
	}

	return subtotal, nil
}
