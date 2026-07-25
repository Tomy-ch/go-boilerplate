// Package summary は、購入集計クエリサービス（query.PurchaseSummaryQueryService）の RDB 実装を提供します。
package summary

import (
	"context"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/uuid"
)

type service struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、購入集計クエリサービスの RDB 実装を生成して返します。
func New(
	db driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.PurchaseSummaryQueryService {
	return &service{
		db:     db,
		tracer: tf.Infra(),
	}
}

// SummarizeByUserID は、認証主体（userID）の購入をステータス単位に集計して購入ステータスマスタの表示順で返します。
// 所有権は WHERE 述語（user_id 一致）で担保するため、他ユーザーの購入は集計に混入しません。
// 購入がない場合は空スライスを返します。
func (s *service) SummarizeByUserID(ctx context.Context, userID uuid.UUID) ([]query.PurchaseStatusSummaryReadModel, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	db := gen.New(driver.New(ctx, s.db))

	rows, err := db.SummarizePurchasesByUserID(ctx, userID)
	if err != nil {
		return nil, pgerror.NormalizeError(err)
	}

	results := make([]query.PurchaseStatusSummaryReadModel, len(rows))
	for i, row := range rows {
		results[i] = query.PurchaseStatusSummaryReadModel{
			StatusID:    row.StatusID,
			StatusName:  row.StatusName,
			Count:       row.PurchaseCount,
			TotalAmount: row.TotalAmount,
		}
	}
	return results, nil
}
