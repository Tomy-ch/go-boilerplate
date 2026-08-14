package cart

import (
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Subtotal は、購入可能な明細だけを合算して決済スケールの整数へ落とします。
// 丸めは合算の後に一度だけ行い、明細ごとの丸め誤差が積み上がらないようにします。
//
// snapshots は商品 ID ごとの観測値で、引けなかった商品は含みません。含まれない明細と、
// 突き合わせで issue が立った明細は合算に入りません。
//
// 幅を超えるのは明細が積み上がった結果に限られます（単価 1 件が決済スケールへ落とせることは
// money.Price の構築時に保証されるため）。合計を偽らずに返す術が無いため、エラーを返します。
func (c *Cart) Subtotal(snapshots map[uuid.UUID]ProductSnapshot) (int64, error) {
	sum := decimal.FromInt(0)

	for _, item := range c.items {
		snapshot, ok := snapshots[item.ProductID()]
		if !ok || !item.Evaluate(&snapshot).HasNoIssue() {
			continue
		}
		sum = sum.Add(snapshot.Price().Decimal().Mul(decimal.FromInt(int64(item.Quantity()))))
	}

	subtotal, err := sum.ToScaledInt64(subtotalMinorUnitDigits)
	if err != nil {
		return 0, xerrors.Join(ErrSubtotalOutOfRange, err)
	}

	return subtotal, nil
}
