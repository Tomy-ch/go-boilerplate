// Package purchase は、購入リポジトリ（purchase.Repository）の RDB 実装を提供します。
package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/domain/kernel/money"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

type repository struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、purchase.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) purchase.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindByID は、ID から購入を明細込みで取得します。存在しない場合は NotFound を返します。
func (r *repository) FindByID(ctx context.Context, id uuid.UUID) (*purchase.Purchase, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	row, err := db.GetPurchaseByID(ctx, id)
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
		details[i] = purchase.NewPurchaseDetail(d.ID, d.ProductID, int(d.Quantity), unitPrice)
	}

	p := row.Purchases
	entity, err := purchase.Reconstruct(
		p.ID,
		p.Code,
		p.UserID,
		p.StatusID,
		int(p.SubtotalAmount),
		int(p.TaxAmount),
		int(p.ShippingFee),
		int(p.TotalAmount),
		details,
		p.OrderedAt,
	)
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}

// FindFeedByUserID は、指定ユーザーの購入履歴を (ordered_at DESC, id DESC) の安定順で
// keyset ページネーション取得します。ステータス名は購入ステータスマスタとの結合で解決します。
// params.AfterOrderedAt / AfterID が nil の場合は先頭ページを、それ以外は境界より過去の行を返します。
func (r *repository) FindFeedByUserID(ctx context.Context, userID uuid.UUID, params purchase.ListFeedParams) ([]purchase.FeedItem, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))

	if params.AfterOrderedAt == nil || params.AfterID == nil {
		rows, err := db.ListPurchasesFeedFirst(ctx, &gen.ListPurchasesFeedFirstParams{
			UserID:     userID,
			LimitParam: params.Limit,
		})
		if err != nil {
			return nil, pgerror.NormalizeError(err)
		}
		items := make([]purchase.FeedItem, len(rows))
		for i, row := range rows {
			items[i] = toFeedItem(row.ID, row.Code, row.TotalAmount, row.OrderedAt, row.StatusID, row.StatusName)
		}
		return items, nil
	}

	rows, err := db.ListPurchasesFeedAfter(ctx, &gen.ListPurchasesFeedAfterParams{
		UserID:         userID,
		AfterOrderedAt: *params.AfterOrderedAt,
		AfterID:        *params.AfterID,
		LimitParam:     params.Limit,
	})
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}
	items := make([]purchase.FeedItem, len(rows))
	for i, row := range rows {
		items[i] = toFeedItem(row.ID, row.Code, row.TotalAmount, row.OrderedAt, row.StatusID, row.StatusName)
	}
	return items, nil
}

// toFeedItem は、購入履歴フィードの行（First / After で別型・同一フィールド）を読み取りモデルへ変換します。
// 合計金額は決済スケール（BIGINT セント）を int へ、ステータス ID / 名称は購入ステータスマスタ由来です。
func toFeedItem(id uuid.UUID, code string, totalAmount int64, orderedAt time.Time, statusID uuid.UUID, statusName string) purchase.FeedItem {
	return purchase.FeedItem{
		Code:        code,
		TotalAmount: int(totalAmount),
		StatusID:    statusID,
		StatusName:  statusName,
		OrderedAt:   orderedAt,
		ID:          id,
	}
}
