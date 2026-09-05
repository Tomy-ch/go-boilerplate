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

// SetItem は、カートへ明細を 1 件置きます。
//
// この本体に外部副作用を足してはなりません。tx がやり直されると二重に実行されます
// （ADR-0035 (transaction-retry-idempotent-callers)）。
func (u *usecase) SetItem(ctx context.Context, params SetItemParams) (CartView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	return tx.DoWithResult(ctx, u.txm, func(ctx context.Context) (CartView, error) {
		return u.setItem(ctx, params)
	})
}

// setItem は、1 トランザクション分の処理です。
func (u *usecase) setItem(ctx context.Context, params SetItemParams) (CartView, error) {
	now := u.clock.Now()

	c, issuedToken, err := u.resolveOrCreateCart(ctx, params.Subject, now)
	if err != nil {
		return CartView{}, err
	}

	if aerr := u.ensureProductAvailable(ctx, params.ProductID); aerr != nil {
		return CartView{}, aerr
	}

	itemID, err := uuid.New()
	if err != nil {
		return CartView{}, xerrors.Wrap(err, "failed to generate cart item id")
	}
	if serr := c.SetItem(cart.SetItemAttributes{
		ItemID:    itemID,
		ProductID: params.ProductID,
		Quantity:  params.Quantity,
	}, now); serr != nil {
		return CartView{}, serr
	}

	view, verr := u.buildView(ctx, c, now)
	if verr != nil {
		return CartView{}, verr
	}
	if uerr := u.cartRepo.Update(ctx, c); uerr != nil {
		return CartView{}, uerr
	}

	view.SessionToken = issuedToken
	return view, nil
}

// ensureProductAvailable は、商品がカートへ入れられる状態にあることを確かめます。
func (u *usecase) ensureProductAvailable(ctx context.Context, productID uuid.UUID) error {
	if _, err := u.productRepo.FindPublishedByID(ctx, productID); err != nil {
		if xerrors.Is(err, apperror.ErrNotFound) {
			return xerrors.Wrap(ErrUnavailableProduct, "product must exist and be published")
		}
		return err
	}
	return nil
}

// resolveOrCreateCart は、主体のカートを永続化済みの状態で返します。カートを持たない主体には作ります。
// 有効期限を過ぎたカートは明細を捨てて空から始めます（docs/spec/usecase/cart.md の SetItem）。
//
// 2 つ目の戻り値は、このときゲストトークンを新しく発行した場合だけ非 nil です。
func (u *usecase) resolveOrCreateCart(
	ctx context.Context, subject Subject, now time.Time,
) (*cart.Cart, *string, error) {
	if subject.UserID != nil {
		return u.resolveOwnerCart(ctx, *subject.UserID, now)
	}
	return u.resolveGuestCart(ctx, subject.SessionToken, now)
}

// resolveOwnerCart は、所有者が確定したカートを解決します。
// ユーザー 1 人につきカートは高々 1 件のため、期限切れでも作り直せません
// （docs/spec/usecase/cart.md の SetItem）。
//
// 解決とロックの間にカートが消えていた場合は、引けなかった場合と同じく確保し直します。
// この op は 404 を宣言していません。
func (u *usecase) resolveOwnerCart(
	ctx context.Context, userID uuid.UUID, now time.Time,
) (*cart.Cart, *string, error) {
	found, err := u.cartRepo.FindByOwnerID(ctx, userID)
	if err != nil {
		if !xerrors.Is(err, apperror.ErrNotFound) {
			return nil, nil, err
		}
		return u.createOwnerCart(ctx, userID, now)
	}

	c, lerr := u.cartRepo.LockByID(ctx, found.ID())
	if lerr != nil {
		if !xerrors.Is(lerr, apperror.ErrNotFound) {
			return nil, nil, lerr
		}
		return u.createOwnerCart(ctx, userID, now)
	}
	if c.IsExpired(now) {
		c.Clear()
	}
	return c, nil, nil
}

// resolveGuestCart は、ゲストのカートを解決します。
// 提示されたトークンで引けなかった場合と、引けたが期限切れだった場合は、どちらも採番し直します
// （docs/spec/usecase/cart.md の SetItem）。
//
// 解決とロックの間にカートが消えていた場合も同じく採番し直します。引き継ぎ（MergeOnLogin）が
// ゲストカートを行ごと消すため、この窓は実際に開きます。この op は 404 を宣言していません。
func (u *usecase) resolveGuestCart(
	ctx context.Context, presented *string, now time.Time,
) (*cart.Cart, *string, error) {
	if presented == nil {
		return u.createGuestCart(ctx, now)
	}

	token, err := cart.NewSessionToken(*presented)
	if err != nil {
		return nil, nil, err
	}

	found, ferr := u.cartRepo.FindBySessionToken(ctx, token)
	if ferr != nil {
		if !xerrors.Is(ferr, apperror.ErrNotFound) {
			return nil, nil, ferr
		}
		return u.createGuestCart(ctx, now)
	}

	c, lerr := u.cartRepo.LockByID(ctx, found.ID())
	if lerr != nil {
		if !xerrors.Is(lerr, apperror.ErrNotFound) {
			return nil, nil, lerr
		}
		return u.createGuestCart(ctx, now)
	}
	if c.IsExpired(now) {
		return u.createGuestCart(ctx, now)
	}
	return c, nil, nil
}

// createOwnerCart は、所有者が確定した空のカートを確保します。
// 一意インデックスが単一文の中で裁定するため、並行して作成が競合しても一意制約違反を上げず、
// 勝ったほうのカートが返ります（MergeOnLogin の引き継ぎ先の確保と同じ扱い）。
// 返った行はその文がロックを取っているので、ここで LockByID は要りません
// （database/dml/repository/cart/insert_owner_cart_if_absent.sql）。
func (u *usecase) createOwnerCart(
	ctx context.Context, userID uuid.UUID, now time.Time,
) (*cart.Cart, *string, error) {
	id, err := uuid.New()
	if err != nil {
		return nil, nil, xerrors.Wrap(err, "failed to generate cart id")
	}

	candidate, nerr := cart.NewForOwner(id, cart.Attributes{OwnerID: &userID, ExpiresAt: now.Add(cartTTL)})
	if nerr != nil {
		return nil, nil, nerr
	}

	c, cerr := u.cartRepo.CreateOwnerIfAbsent(ctx, candidate)
	if cerr != nil {
		return nil, nil, cerr
	}
	if c.IsExpired(now) {
		c.Clear()
	}
	return c, nil, nil
}

// createGuestCart は、新しいトークンを発行して空のゲストカートを作り、そのトークンを返します。
func (u *usecase) createGuestCart(ctx context.Context, now time.Time) (*cart.Cart, *string, error) {
	raw, err := u.tokenGen.Generate()
	if err != nil {
		return nil, nil, xerrors.Wrap(err, "failed to generate session token")
	}

	token, terr := cart.NewSessionToken(raw)
	if terr != nil {
		return nil, nil, terr
	}

	id, ierr := uuid.New()
	if ierr != nil {
		return nil, nil, xerrors.Wrap(ierr, "failed to generate cart id")
	}

	c, nerr := cart.NewForGuest(id, token, now.Add(cartTTL))
	if nerr != nil {
		return nil, nil, nerr
	}
	if cerr := u.cartRepo.Create(ctx, c); cerr != nil {
		return nil, nil, cerr
	}
	return c, &raw, nil
}
