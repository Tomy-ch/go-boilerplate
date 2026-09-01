// Package purchasestatus は、購入ステータスリポジトリ（status.Repository）の RDB 実装を提供します。
package purchasestatus

import (
	"context"

	"go-boilerplate/internal/domain/purchase/status"
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

// New は、status.Repository の RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) status.Repository {
	return &repository{
		db:     db,
		tracer: tf.Infra(),
	}
}

// FindAll は、全購入ステータスエンティティを sortKey 昇順で取得します。
func (r *repository) FindAll(ctx context.Context) (status.Statuses, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.GetPurchaseStatusDomainAll(ctx)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	purchaseStatuses := make(status.Statuses, len(rows))
	for i, row := range rows {
		entity, err := rowToPurchaseStatus(row.ID, row.Name, row.Code, row.SortKey)
		if err != nil {
			return nil, err
		}
		purchaseStatuses[i] = entity
	}

	return purchaseStatuses, nil
}

// rowToPurchaseStatus は、DB 行の値からドメインエンティティを再構築します。
func rowToPurchaseStatus(id uuid.UUID, name string, code, sortKey int16) (*status.Status, error) {
	entity, err := status.New(id, status.Attributes{Name: name, Code: int(code), SortKey: int(sortKey)})
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}
