package purchase

import (
	"context"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/pkg/uuid"
)

// FindStatusesByProductID は、指定商品を明細に持つ購入が取っているステータスを重複なく取得します。
// 進行中かどうかでは絞り込まず、code は購入ステータスマスタとの結合で解決してドメインの値へ復元します。
func (r *repository) FindStatusesByProductID(ctx context.Context, productID uuid.UUID) ([]purchase.Status, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	codes, err := db.SelectPurchaseStatusCodesByProductID(ctx, productID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	statuses := make([]purchase.Status, len(codes))
	for i, code := range codes {
		status, serr := purchase.NewStatus(int(code))
		if serr != nil {
			return nil, serr
		}
		statuses[i] = status
	}

	return statuses, nil
}
