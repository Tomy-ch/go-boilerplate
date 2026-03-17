// Package healthcheck は、システムの健全性チェックに関するシステムクエリを提供します。
package healthcheck

import (
	"context"
	"time"

	"boilerplate-go/internal/infrastructure/rdb/driver/loggingdb"
	"boilerplate-go/internal/infrastructure/rdb/postgres/pgerror"
	"boilerplate-go/internal/infrastructure/rdb/sqlc/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/healthcheck/query"
)

type systemQuery struct {
	db     loggingdb.DBProvider
	tracer observability.LayerTracer
}

func New(provider loggingdb.DBProvider, tf observability.TracerFactory) query.DBSystemQuery {
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
	latency := time.Since(start)

	return query.DBHealth{
		Ready:       true,
		ResponsedAt: time.Now(),
		Latency:     latency,
	}, nil
}
