package cart

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/xerrors"
)

// ClearCart は、再評価を行いません。応答は本文を持たず価格を 1 つも提示していないため、
// 提示価格を記録すると、次の取得で立つはずの priceIncreased を消してしまいます
// （docs/spec/cart/usecase.md の ClearCart）。
//
// カートの行は消しません。消すと直後の操作でセッショントークンが発行し直され、利用者の
// 同一性が切れます。
func (u *usecase) ClearCart(ctx context.Context, subject Subject) error {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()
	return u.txm.Do(ctx, func(ctx context.Context) error {
		c, found, err := u.resolveCart(ctx, subject, now)
		if err != nil {
			return err
		}
		if !found {
			return nil
		}

		locked, lerr := u.cartRepo.LockByID(ctx, c.ID())
		if lerr != nil {
			// 解決とロックの間にカートが消えていても、カートを持たない主体と同じく成功で返します。
			// 集約の不在は 1 つの事実であり、どちらの読み取りで見つけたかで応答を変えません
			// （この op は 404 を宣言していません）。
			if xerrors.Is(lerr, apperror.ErrNotFound) {
				return nil
			}
			return lerr
		}

		locked.Clear()
		locked.Touch(now, cartTTL)
		return u.cartRepo.Update(ctx, locked)
	})
}
