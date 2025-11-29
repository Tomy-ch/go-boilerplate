// Package healthcheck は、システムの健全性チェックに関するシステムクエリを提供します。
package healthcheck

import (
	"context"
	"database/sql"
	"time"

	"boilerplate-go/internal/apperror"
	rdbdriver "boilerplate-go/internal/infrastructure/rdb/driver"
	gen "boilerplate-go/internal/infrastructure/rdb/sqlc"
	"boilerplate-go/internal/usecase/healthcheck/query"
	"boilerplate-go/pkg/xerrors"

	"go.uber.org/zap"
)

type systemQuery struct {
	db *sql.DB
	z  *zap.Logger
}

func New(db *sql.DB, z *zap.Logger) query.DBSystemQuery {
	return &systemQuery{
		db: db,
		z:  z,
	}
}

// CheckDBHealth は、データベースの健全性をチェックします。
func (s *systemQuery) CheckDBHealth(ctx context.Context) (query.DBHealth, error) {
	start := time.Now()
	db := gen.New(rdbdriver.ResolveDriverWithLog(ctx, s.db, s.z))
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
