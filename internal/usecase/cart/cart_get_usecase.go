package cart

import (
	"context"

	"go-boilerplate/internal/usecase/boundary/tx"
)

// GetCart は、取得が書き込みを伴います。値上がりの判定には前回提示した価格が要るため提示のたびに
// 記録し、あわせて有効期限を延長するためです。したがってこの取得はトランザクションを要求します。
//
// この本体に外部副作用を足してはなりません。tx がシリアライズ失敗・デッドロック検出で再実行します
// （ADR-0033 (transaction-retry-idempotent-callers)）。
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

		view, verr := u.buildView(ctx, c, now)
		if verr != nil {
			return CartView{}, verr
		}
		if uerr := u.cartRepo.Update(ctx, c); uerr != nil {
			return CartView{}, uerr
		}

		return view, nil
	})
}
