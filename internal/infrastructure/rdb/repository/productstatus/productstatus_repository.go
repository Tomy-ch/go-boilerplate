// Package productstatus は、商品ステータスリポジトリ（status.Repository）の RDB 実装を提供します。
package productstatus

import (
	"context"

	"go-boilerplate/internal/domain/product/status"
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

// FindAll は、全商品ステータスエンティティを sortKey 昇順で取得します。
func (r *repository) FindAll(ctx context.Context) (status.Statuses, error) {
	ctx, endSpan := r.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, r.db))
	rows, err := db.GetProductStatusDomainAll(ctx)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	productStatuses := make(status.Statuses, len(rows))
	for i, row := range rows {
		entity, err := rowToProductStatus(row.ID, row.Name, row.Code, row.SortKey)
		if err != nil {
			return nil, err
		}
		productStatuses[i] = entity
	}

	return productStatuses, nil
}

// rowToProductStatus は、DB 行の値からドメインエンティティを再構築します。
// 再構築時の検証失敗はデータ不整合として ErrInternal へ正規化します（422 / details にしない）。
func rowToProductStatus(id uuid.UUID, name string, code, sortKey int16) (*status.Status, error) {
	entity, err := status.New(id, name, int(code), int(sortKey))
	if err != nil {
		return nil, pgerror.NormalizeReconstructError(err)
	}
	return entity, nil
}
