// Package healthcheck は、システムの健全性チェックに関するシステムクエリを提供します。
package healthcheck

import (
	"context"
	"database/sql"
	"time"

	"boilerplate-go/internal/apperror"
	rdbdriver "boilerplate-go/internal/infrastructure/rdb/driver"
	"boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/healthcheck/query"
	"boilerplate-go/pkg/xerrors"

	"go.uber.org/zap"
)

type systemQuery struct {
	tracer observability.LayerTracer
	db     *sql.DB
	z      *zap.Logger
}

func New(db *sql.DB, z *zap.Logger, tf observability.TracerFactory) query.DBSystemQuery {
	return &systemQuery{
		tracer: tf.Infra(),
		db:     db,
		z:      z,
	}
}

// CheckDBHealth は、データベースの健全性をチェックします。
func (s *systemQuery) CheckDBHealth(ctx context.Context) (query.DBHealth, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	start := time.Now()
	db := sqlc.New(rdbdriver.ResolveDriverWithLog(ctx, s.db, s.z))
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
