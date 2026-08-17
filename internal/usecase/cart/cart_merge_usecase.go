package cart

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// MergeOnLogin は、引き継ぎ先のカートを先に確保してから、所有権を評価する前に両方をロックします
// （ADR-0035 (ordered-pessimistic-row-locks)）。
//
// 確保とロックはいずれも競合しない単一の呼び出しで行い、やり直しの機構をどこにも置きません。
// 分割するとトランザクションごと中断するため、Repository の CreateOwnerIfAbsent と LockByIDs を
// 存在確認や 1 件ずつのロックへ置き換えてはなりません。
func (u *usecase) MergeOnLogin(ctx context.Context, params MergeOnLoginParams) (MergeCartResult, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (MergeCartResult, error) {
		guestID, found, err := u.findGuestCartID(ctx, params.SessionToken, now)
		if err != nil || !found {
			return MergeCartResult{}, err
		}

		ownerID, oerr := u.ensureOwnerCartID(ctx, params.UserID, now)
		if oerr != nil {
			return MergeCartResult{}, oerr
		}

		locked, lerr := u.cartRepo.LockByIDs(ctx, []uuid.UUID{guestID, ownerID})
		if lerr != nil {
			return MergeCartResult{}, lerr
		}

		byID := indexCarts(locked)
		guest, owner := byID[guestID], byID[ownerID]
		if guest == nil || owner == nil {
			// 解決とロックの間にどちらかが消えた。失われるものは無いので成功で返します。
			return MergeCartResult{}, nil
		}

		return u.mergeInto(ctx, owner, guest, now)
	})
}

// findGuestCartID は、引き継ぎ元のゲストカートの ID を引きます。
// found=false は引き継ぐものが無いことを表し、失敗ではありません。所有者が確定したカートは
// トークンを持たないため引き継ぎ済みのカートも、期限切れのカートも、ここで found=false になります。
func (u *usecase) findGuestCartID(
	ctx context.Context, sessionToken string, now time.Time,
) (uuid.UUID, bool, error) {
	token, err := cart.NewSessionToken(sessionToken)
	if err != nil {
		return uuid.UUID{}, false, err
	}

	guest, ferr := u.cartRepo.FindBySessionToken(ctx, token)
	if ferr != nil {
		if xerrors.Is(ferr, apperror.ErrNotFound) {
			return uuid.UUID{}, false, nil
		}
		return uuid.UUID{}, false, ferr
	}
	if guest.IsExpired(now) {
		return uuid.UUID{}, false, nil
	}
	return guest.ID(), true, nil
}

// ensureOwnerCartID は、引き継ぎ先のカートを確保してその ID を返します。
func (u *usecase) ensureOwnerCartID(
	ctx context.Context, userID uuid.UUID, now time.Time,
) (uuid.UUID, error) {
	id, err := uuid.New()
	if err != nil {
		return uuid.UUID{}, xerrors.Wrap(err, "failed to generate cart id")
	}

	candidate, nerr := cart.NewForOwner(id, cart.Attributes{
		OwnerID:   &userID,
		ExpiresAt: now.Add(cartTTL),
	})
	if nerr != nil {
		return uuid.UUID{}, nerr
	}

	owner, cerr := u.cartRepo.CreateOwnerIfAbsent(ctx, candidate)
	if cerr != nil {
		return uuid.UUID{}, cerr
	}
	return owner.ID(), nil
}

// mergeInto は、ゲストカートの明細を引き継ぎ先へ取り込み、ゲストカートを破棄します。
//
// 破棄は保存より先に行います。取り込んだ明細は ID を保ったまま引き継ぎ先へ移るため、引き継ぎ元の
// 行が残ったまま保存すると明細の主キーが衝突します。順序を入れ替えてはなりません。
// 2 つの書き込みは呼び出し元のトランザクションの中にあり、保存に失敗すれば破棄ごと巻き戻ります。
//
// 引き継ぎ元は行ごと消えるため、元のセッショントークンでそのカートへ到達する経路は状態として
// 残りません。
func (u *usecase) mergeInto(
	ctx context.Context, owner, guest *cart.Cart, now time.Time,
) (MergeCartResult, error) {
	result := owner.Merge(guest, now)
	owner.Touch(now, cartTTL)

	if err := u.cartRepo.Delete(ctx, guest.ID()); err != nil {
		return MergeCartResult{}, err
	}
	if err := u.cartRepo.Update(ctx, owner); err != nil {
		return MergeCartResult{}, err
	}

	return MergeCartResult{Clamped: result.Clamped(), Dropped: result.Dropped()}, nil
}

// indexCarts は、ロックで得たカートを ID で引ける表にします。
// LockByIDs は id 昇順で返すため、位置ではなく ID で引き当てます。
func indexCarts(carts cart.Carts) map[uuid.UUID]*cart.Cart {
	byID := make(map[uuid.UUID]*cart.Cart, len(carts))
	for _, c := range carts {
		byID[c.ID()] = c
	}
	return byID
}
