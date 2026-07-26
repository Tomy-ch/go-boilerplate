// Package healthcheck は、システムの健全性チェックに関するシステムクエリを提供します。
package healthcheck

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/healthcheck/query"
)

type systemQuery struct {
	db     driver.DatabaseDriver
	tracer observability.LayerTracer
}

// New は、DB ヘルスチェック用のシステムクエリ実装を生成して返します。
func New(
	provider driver.DatabaseDriver,
	tf observability.TracerFactory,
) query.DBSystemCqrs {
	return &systemQuery{
		db:     provider,
		tracer: tf.Infra(),
	}
}

// CheckDBHealth は、軽量クエリを 1 本発行して往復レイテンシを計測し、応答があれば Ready=true と
// 応答時刻・レイテンシを返します。
func (s *systemQuery) CheckDBHealth(ctx context.Context) (query.DBHealth, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	start := time.Now()
	db := gen.New(driver.New(ctx, s.db))
	_, err := db.GetDBHealthCheck(ctx)
	if err != nil {
		return query.DBHealth{}, pgerror.NormalizeError(err)
	}
	respondedAt := time.Now()

	return query.DBHealth{
		Ready:       true,
		RespondedAt: respondedAt,
		Latency:     respondedAt.Sub(start),
	}, nil
}
