package cart

import (
	"context"

	"go-boilerplate/internal/usecase/boundary/tx"
)

// GetCart は、主体のカートを明細ごとの再評価つきで返します。
//
// 取得が書き込みを伴います。値上がりの判定には前回提示した価格が要るため提示のたびに記録し、
// あわせて有効期限を延長するためです。したがってこの取得はトランザクションを要求します。
//
// tx はシリアライズ失敗・デッドロック検出で本体を再実行しますが、この本体は「読む・判定する・
// 同じ導出値を書く」だけで外部副作用を持たないため、何回走らせても結果が変わりません
// （ADR-0033 (transaction-retry-idempotent-callers) が呼び出し側へ課す冪等性の制約を満たします）。
func (u *usecase) GetCart(ctx context.Context, subject Subject) (CartView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (CartView, error) {
		c, found, err := u.resolveCart(ctx, subject, now)
		if err != nil {
			return CartView{}, err
		}
		if !found {
			return emptyCartView(), nil
		}

		items := c.Items()
		products, ferr := u.findProducts(ctx, items)
		if ferr != nil {
			return CartView{}, ferr
		}

		// 判定も合算も提示価格を書き換える前に済ませる。書き換えた後では、値上がりが常に
		// 「差が無い」になり、除外されるはずの明細が小計に入る。
		views, seen := evaluateItems(items, products)
		subtotal, serr := c.Subtotal(toSnapshots(products))
		if serr != nil {
			return CartView{}, serr
		}

		c.MarkSeen(seen)
		c.Touch(now, cartTTL)
		if uerr := u.cartRepo.Update(ctx, c); uerr != nil {
			return CartView{}, uerr
		}

		expiresAt := c.ExpiresAt()
		return CartView{Items: views, SubtotalAmount: subtotal, ExpiresAt: &expiresAt}, nil
	})
}
