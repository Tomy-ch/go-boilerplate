// Package healthcheck は、システムの健全性チェックに関するシステムクエリを提供します。
package healthcheck

import (
	"context"
	"time"

	"go-boilerplate/internal/infrastructure/rdb/driver/loggingdb"
	"go-boilerplate/internal/infrastructure/rdb/pgerror"
	"go-boilerplate/internal/infrastructure/rdb/sqlc/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/healthcheck/query"
)

type systemQuery struct {
	db     loggingdb.DBProvider
	tracer observability.LayerTracer
}

func New(
	provider loggingdb.DBProvider,
	tf observability.TracerFactory,
) query.DBSystemQuery {
	return &systemQuery{
		db:     provider,
		tracer: tf.Infra(),
	}
}

// CheckDBHealth は、データベースの健全性をチェックします。
func (s *systemQuery) CheckDBHealth(ctx context.Context) (query.DBHealth, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	start := time.Now()
	db := gen.New(s.db.NewLoggingDB(ctx))
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
