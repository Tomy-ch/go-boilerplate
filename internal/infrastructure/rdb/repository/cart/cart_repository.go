// Package cart は、カートリポジトリ（cart.Repository）の RDB 実装を提供します。
package cart

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/cart"
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、cart.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) cart.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindBySessionToken は、セッショントークンからカートを明細込みで再構築して返します。
// 存在しない場合は NotFound を返します。
func (r *repository) FindBySessionToken(ctx context.Context, token cart.SessionToken) (*cart.Cart, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	value := token.Value()
	row, err := db.GetCartBySessionToken(ctx, &value)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return r.reconstruct(ctx, db, row.Carts)
}

// FindByOwnerID は、所有者からカートを明細込みで再構築して返します。
// 存在しない場合は NotFound を返します。
func (r *repository) FindByOwnerID(ctx context.Context, userID uuid.UUID) (*cart.Cart, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.GetCartByOwnerID(ctx, &userID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return r.reconstruct(ctx, db, row.Carts)
}

// LockByID は、カート行を悲観ロック（FOR UPDATE）して明細込みで再構築して返します。
// 存在しない場合は NotFound を返します。
func (r *repository) LockByID(ctx context.Context, id uuid.UUID) (*cart.Cart, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.LockCartByID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return r.reconstruct(ctx, db, row.Carts)
}

// LockByIDs は、カート行を ID 昇順にまとめて悲観ロック（FOR UPDATE）し、明細込みで再構築して返します。
// 不存在の ID は結果に現れないため、返る件数は引数より少なくなり得ます。
func (r *repository) LockByIDs(ctx context.Context, ids []uuid.UUID) (cart.Carts, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	rows, err := db.LockCartsByIDs(ctx, ids)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	carts := make(cart.Carts, 0, len(rows))
	for _, row := range rows {
		entity, rerr := r.reconstruct(ctx, db, row.Carts)
		if rerr != nil {
			return nil, rerr
		}
		carts = append(carts, entity)
	}
	return carts, nil
}

// CreateOwnerIfAbsent は、所有者のカートが無ければ空のカートを 1 件登録し、確定した行を返します。
// 一意インデックスが単一文の中で裁定するため、既にある場合も一意制約違反を上げず、既存の行が返ります。
// 明細は書きません（作るのは空のカートだけです）。
func (r *repository) CreateOwnerIfAbsent(ctx context.Context, c *cart.Cart) (*cart.Cart, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.CreateOwnerCartIfAbsent(ctx, &gen.CreateOwnerCartIfAbsentParams{
		ID:        c.ID(),
		UserID:    c.OwnerID(),
		ExpiresAt: c.ExpiresAt(),
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	return r.reconstruct(ctx, db, row.Carts)
}

// Create は、カートを明細込みで新規登録します。
// user_id / session_token の一意制約違反は Conflict へ正規化されます。
func (r *repository) Create(ctx context.Context, c *cart.Cart) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	if err := db.CreateCart(ctx, &gen.CreateCartParams{
		ID:           c.ID(),
		UserID:       c.OwnerID(),
		SessionToken: sessionTokenValue(c),
		ExpiresAt:    c.ExpiresAt(),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}

	for _, item := range c.Items() {
		quantity, err := safecast.IntToInt32(item.Quantity())
		if err != nil {
			return xerrors.Wrap(err, "invalid cart item quantity")
		}
		if err := db.CreateCartItem(ctx, &gen.CreateCartItemParams{
			ID:            item.ID(),
			CartID:        c.ID(),
			ProductID:     item.ProductID(),
			Quantity:      quantity,
			LastSeenPrice: lastSeenPriceValue(item),
			AddedAt:       item.AddedAt(),
		}); err != nil {
			return pgerror.NormalizeError(err)
		}
	}
	return nil
}

// Update は、カートを明細込みで現在の状態へ一致させます。存在しない場合は NotFound を返します。
//
// 複数の文に分かれるため、呼び出し元は ctx にトランザクションを載せてください
// （internal/infrastructure/rdb/README.md の Centralized Transaction Boundary Management）。
// 境界が無いまま呼ぶと、親行だけが確定した状態が残りえます。
func (r *repository) Update(ctx context.Context, c *cart.Cart) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	affected, err := db.UpdateCart(ctx, &gen.UpdateCartParams{
		UserID:       c.OwnerID(),
		SessionToken: sessionTokenValue(c),
		ExpiresAt:    c.ExpiresAt(),
		ID:           c.ID(),
	})
	if err := pgerror.NormalizeExecResult(affected, err); err != nil {
		return err
	}

	items := c.Items()
	productIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		quantity, cerr := safecast.IntToInt32(item.Quantity())
		if cerr != nil {
			return xerrors.Wrap(cerr, "invalid cart item quantity")
		}
		if uerr := db.UpsertCartItem(ctx, &gen.UpsertCartItemParams{
			ID:            item.ID(),
			CartID:        c.ID(),
			ProductID:     item.ProductID(),
			Quantity:      quantity,
			LastSeenPrice: lastSeenPriceValue(item),
			AddedAt:       item.AddedAt(),
		}); uerr != nil {
			return pgerror.NormalizeError(uerr)
		}
		productIDs = append(productIDs, item.ProductID())
	}

	if derr := db.DeleteCartItemsNotIn(ctx, &gen.DeleteCartItemsNotInParams{
		CartID:     c.ID(),
		ProductIds: productIDs,
	}); derr != nil {
		return pgerror.NormalizeError(derr)
	}
	return nil
}

// Delete は、カートを明細ごと削除します。存在しない場合もエラーとしません。
func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	if err := db.DeleteCart(ctx, id); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// DeleteExpired は、有効期限を過ぎたカートを最大 limit 件削除し、削除件数を返します。
func (r *repository) DeleteExpired(ctx context.Context, now time.Time, limit int32) (int, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	affected, err := db.DeleteExpiredCarts(ctx, &gen.DeleteExpiredCartsParams{
		Now:      now,
		RowLimit: limit,
	})
	if err != nil {
		return 0, pgerror.NormalizeError(err)
	}

	// 削除件数は limit（int32）を超えないため、int への変換で失われる桁はない。
	return int(affected), nil
}

// reconstruct は、カート行と明細行から集約を再構築します。
func (r *repository) reconstruct(ctx context.Context, db *gen.Queries, row gen.Carts) (*cart.Cart, error) {
	itemRows, err := db.ListCartItemsByCartID(ctx, row.ID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	items, err := toCartItems(itemRows)
	if err != nil {
		return nil, err
	}

	token, err := toSessionToken(row.SessionToken)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}

	entity, err := cart.Reconstruct(row.ID, cart.Attributes{
		OwnerID:      row.UserID,
		SessionToken: token,
		Items:        items,
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	})
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// toCartItems は、明細行を値オブジェクトへ変換します。
func toCartItems(rows []*gen.ListCartItemsByCartIDRow) ([]cart.CartItem, error) {
	items := make([]cart.CartItem, 0, len(rows))
	for _, row := range rows {
		ci := row.CartItems

		price, err := toPrice(ci.LastSeenPrice)
		if err != nil {
			return nil, pgerror.NormalizeReconstructError(err)
		}

		items = append(items, cart.NewCartItem(ci.ID, cart.CartItemAttributes{
			ProductID:     ci.ProductID,
			Quantity:      int(ci.Quantity),
			AddedAt:       ci.AddedAt,
			LastSeenPrice: price,
		}))
	}
	return items, nil
}

// toSessionToken は、列の値をセッショントークンへ変換します。NULL の場合は nil を返します。
func toSessionToken(value *string) (*cart.SessionToken, error) {
	if value == nil {
		return nil, nil //nolint:nilnil // 所有者が確定したカートはトークンを持たない
	}
	token, err := cart.NewSessionToken(*value)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// toPrice は、列の値を価格へ変換します。NULL の場合は nil を返します。
func toPrice(value *decimal.Decimal) (*money.Price, error) {
	if value == nil {
		return nil, nil //nolint:nilnil // 未提示の価格は NULL で表される
	}
	price, err := money.NewPrice(*value)
	if err != nil {
		return nil, err
	}
	return &price, nil
}

// sessionTokenValue は、集約のセッショントークンを列の値へ変換します。
func sessionTokenValue(c *cart.Cart) *string {
	token := c.SessionToken()
	if token == nil {
		return nil
	}
	value := token.Value()
	return &value
}

// lastSeenPriceValue は、明細の提示価格を列の値へ変換します。
func lastSeenPriceValue(item cart.CartItem) *decimal.Decimal {
	price := item.LastSeenPrice()
	if price == nil {
		return nil
	}
	value := price.Decimal()
	return &value
}
