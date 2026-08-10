// Package purchase は、購入 CommandService（command.CommandService）の RDB 実装を提供します。
// 在庫減算・購入・明細の書き込みを、渡された ctx のトランザクション内で原子的に実行します。
package purchase

import (
	"context"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/command"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

type commandService struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、purchase の CommandService の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) command.CommandService {
	return &commandService{
		db:     db,
		tracer: tf.Infra(),
	}
}

// CreatePurchase は、在庫減算・purchases INSERT・purchase_details INSERT を渡された tx 内で原子的に実行します。
// 在庫減算は防御的に売り越しを弾き、更新 0 行の場合は ErrInsufficientStock（409）を返します。
// この防御は domain の売り越し判定を言い換えた fail-closed の二重防御であり、独立した規則ではありません
// （ADR-0029 (lightweight-cqrs) § Derivation）。返すエラーも domain の sentinel をそのまま用います。
func (c *commandService) CreatePurchase(ctx context.Context, p *purchase.Purchase) error {
	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, c.db))
	details := p.Details()

	quantities, err := toDetailQuantities(details)
	if err != nil {
		return err
	}
	statusCode, err := safecast.IntToInt16(p.StatusCode())
	if err != nil {
		return xerrors.Wrap(err, "invalid purchase status code")
	}

	for i, d := range details {
		affected, derr := db.DecrementProductStock(ctx, &gen.DecrementProductStockParams{
			QuantityParam:  quantities[i],
			ProductIDParam: d.ProductID(),
		})
		if derr != nil {
			return pgerror.NormalizeError(derr)
		}
		if affected == 0 {
			return purchase.ErrInsufficientStock
		}
	}

	if err := db.InsertPurchase(ctx, &gen.InsertPurchaseParams{
		ID:             p.ID(),
		Code:           p.Code(),
		UserID:         p.UserID(),
		StatusCode:     statusCode,
		SubtotalAmount: int64(p.SubtotalAmount()),
		TaxAmount:      int64(p.TaxAmount()),
		ShippingFee:    int64(p.ShippingFee()),
		TotalAmount:    int64(p.TotalAmount()),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}

	for i, d := range details {
		if err := db.InsertPurchaseDetail(ctx, &gen.InsertPurchaseDetailParams{
			ID:         d.ID(),
			PurchaseID: p.ID(),
			ProductID:  d.ProductID(),
			Quantity:   quantities[i],
			UnitPrice:  d.UnitPrice().Decimal(),
		}); err != nil {
			return pgerror.NormalizeError(err)
		}
	}
	return nil
}

// LockPurchase は、購入行のみ悲観ロック（FOR UPDATE OF p）して明細込みで再構築し返します。
// キャンセルの状態遷移の競合（同一購入への並行キャンセル）を購入行ロックで直列化します。
// 現在状態は購入ステータスマスタとの結合で code を解決します。存在しない場合は NotFound を返します。
func (c *commandService) LockPurchase(ctx context.Context, id uuid.UUID) (*purchase.Purchase, error) {
	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, c.db))

	row, err := db.GetPurchaseByIDForUpdate(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	detailRows, err := db.ListPurchaseDetailsByPurchaseID(ctx, id)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	details := make([]purchase.PurchaseDetail, len(detailRows))
	for i, dr := range detailRows {
		d := dr.PurchaseDetails
		unitPrice, perr := money.NewPrice(d.UnitPrice)
		if perr != nil {
			return nil, pgerror.NormalizeReconstructError(perr)
		}
		details[i] = purchase.NewPurchaseDetail(d.ID, purchase.PurchaseDetailAttributes{
			ProductID: d.ProductID,
			Quantity:  int(d.Quantity),
			UnitPrice: unitPrice,
		})
	}

	p := row.Purchases
	entity, err := purchase.Reconstruct(p.ID, purchase.Attributes{
		Code:           p.Code,
		UserID:         p.UserID,
		StatusID:       p.StatusID,
		StatusCode:     int(row.StatusCode),
		SubtotalAmount: int(p.SubtotalAmount),
		TaxAmount:      int(p.TaxAmount),
		ShippingFee:    int(p.ShippingFee),
		TotalAmount:    int(p.TotalAmount),
		Details:        details,
		OrderedAt:      p.OrderedAt,
		PaidAt:         p.PaidAt,
		CanceledAt:     p.CanceledAt,
		ShippedAt:      p.ShippedAt,
		DeliveredAt:    p.DeliveredAt,
	})
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// CancelPurchase は、キャンセルに伴う在庫復元（明細分の加算）と購入の状態更新（status_id / canceled_at）を
// 渡された tx 内で原子的に実行します。在庫加算は相対更新で売り越しを生まないため在庫不足ガードは不要です。
func (c *commandService) CancelPurchase(ctx context.Context, p *purchase.Purchase) error {
	ctx, endSpan := c.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, c.db))
	details := p.Details()

	quantities, err := toDetailQuantities(details)
	if err != nil {
		return err
	}
	statusCode, err := safecast.IntToInt16(p.StatusCode())
	if err != nil {
		return xerrors.Wrap(err, "invalid purchase status code")
	}

	for i, d := range details {
		affected, ierr := db.IncrementProductStock(ctx, &gen.IncrementProductStockParams{
			QuantityParam:  quantities[i],
			ProductIDParam: d.ProductID(),
		})
		// 対象商品が不存在（影響 0 行）なら fail-closed で NotFound（減算側の売り越しガードと対称の二重防御）。
		if nerr := pgerror.NormalizeExecResult(affected, ierr); nerr != nil {
			return nerr
		}
	}

	if err := db.UpdatePurchaseCanceled(ctx, &gen.UpdatePurchaseCanceledParams{
		StatusCode: statusCode,
		CanceledAt: p.CanceledAt(),
		ID:         p.ID(),
	}); err != nil {
		return pgerror.NormalizeError(err)
	}
	return nil
}

// toDetailQuantities は、購入明細の数量を sqlc の int32（DB INTEGER 列）へ明細の順序どおりに変換します。
// 数量が INTEGER 列に収まらない明細が 1 つでもあれば、部分的な書き込みを避けるため変換全体を失敗させます。
func toDetailQuantities(details []purchase.PurchaseDetail) ([]int32, error) {
	quantities := make([]int32, len(details))
	for i, d := range details {
		q, err := safecast.IntToInt32(d.Quantity())
		if err != nil {
			return nil, xerrors.Wrap(err, "invalid purchase detail quantity")
		}
		quantities[i] = q
	}
	return quantities, nil
}
