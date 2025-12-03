//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package healthcheck は、システムの健全性チェックに関するユースケースを提供します。
package healthcheck

import (
	"context"
	"time"

	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/usecase/healthcheck/query"
)

const (
	Degraded  = "degraded"
	Ok        = "ok"
	Unhealthy = "unhealthy"
)

// DTO は、システムの健全性に関するデータ転送用のオブジェクトです。
type DTO struct {
	Status          string
	ApplicationTime time.Time
	DBHealthCheck   query.DBHealth
}

// usecase は、システムの健全性チェックに関するユースケースを提供します。
type usecase struct {
	tracer        observability.LayerTracer
	dbSystemQuery query.DBSystemQuery
}

// Usecase は、システムの健全性チェックに関するユースケースを定義します。
type Usecase interface {
	CheckHealth(ctx context.Context) (DTO, error)
}

// New は、システムの健全性チェックに関するユースケースを初期化します。
func New(dbsq query.DBSystemQuery, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer:        tf.Usecase(),
		dbSystemQuery: dbsq,
	}
}

// CheckHealth は、システムの健全性をチェックするユースケースです。
func (u *usecase) CheckHealth(ctx context.Context) (DTO, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	applTime := time.Now()
	dbHealth, err := u.dbSystemQuery.CheckDBHealth(ctx)
	if err != nil {
		return DTO{
			Status:          Unhealthy,
			ApplicationTime: applTime,
		}, err
	}

	return DTO{
		Status:          Ok,
		ApplicationTime: applTime,
		DBHealthCheck:   dbHealth,
	}, nil
}
