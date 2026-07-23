// Package purchase は、購入リポジトリ（purchase.Repository）の RDB 実装を提供します。
package purchase

import (
	"context"

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
		details[i] = purchase.NewPurchaseDetail(d.ID, d.ProductID, int(d.Quantity), int(d.UnitPrice))
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
