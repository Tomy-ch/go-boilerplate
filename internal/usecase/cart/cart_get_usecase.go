package cart

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/xerrors"
)

// GetCart は、取得が書き込みを伴います。値上がりの判定には前回提示した価格が要るため提示のたびに
// 記録し、あわせて有効期限を延長するためです。したがってこの取得はトランザクションを要求します。
//
// この本体に外部副作用を足してはなりません。tx がシリアライズ失敗・デッドロック検出で再実行します
// （ADR-0034 (transaction-retry-idempotent-callers)）。
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

		// 解決はロックを取らないため、その結果をそのまま書き戻してはなりません。書き込みは集約単位で
		// 明細集合を丸ごと反映するため、解決から書き戻しまでの間に他の操作が明細を消していた場合、
		// 古い集約を書き戻すとそれが復活します。書き込む対象はロックで取り直したものだけです。
		locked, lerr := u.cartRepo.LockByID(ctx, c.ID())
		if lerr != nil {
			if xerrors.Is(lerr, apperror.ErrNotFound) {
				return emptyCartView(), nil
			}
			return CartView{}, lerr
		}

		view, verr := u.buildView(ctx, locked, now)
		if verr != nil {
			return CartView{}, verr
		}
		if uerr := u.cartRepo.Update(ctx, locked); uerr != nil {
			return CartView{}, uerr
		}

		return view, nil
	})
}
