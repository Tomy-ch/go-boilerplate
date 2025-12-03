// Package healthcheck は、システムの健全性チェックに関するシステムクエリを提供します。
package healthcheck

import (
	"context"
	"time"

	"boilerplate-go/internal/apperror"
	"boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/healthcheck/query"
	"boilerplate-go/pkg/xerrors"
)

type systemQuery struct {
	db       driver.DatabaseDriver
	provider driver.LoggingDBProvider
	tracer   observability.LayerTracer
}

func New(db driver.DatabaseDriver, provider driver.LoggingDBProvider, tf observability.TracerFactory) query.DBSystemQuery {
	return &systemQuery{
		db:       db,
		provider: provider,
		tracer:   tf.Infra(),
	}
}

// CheckDBHealth は、データベースの健全性をチェックします。
func (s *systemQuery) CheckDBHealth(ctx context.Context) (query.DBHealth, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	start := time.Now()
	db := sqlc.New(s.provider.NewLoggingDB(ctx))
	_, err := db.GetDBHealthCheck(ctx)
	if err != nil {
		return query.DBHealth{}, xerrors.Wrap(apperror.ErrUnavailable, err.Error())
	}
	latency := time.Since(start)

	return query.DBHealth{
		Ready:       true,
		ResponsedAt: time.Now(),
		Latency:     latency,
	}, nil
}
