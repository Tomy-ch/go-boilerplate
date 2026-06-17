//go:generate mockgen -source=$GOFILE -destination=mock/mock_health_check_usecase.gen.go -package=mock_$GOPACKAGE

// Package healthcheck は、システムの健全性チェックに関するユースケースを提供します。
package healthcheck

import (
	"context"
	"time"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/healthcheck/query"
)

// ヘルスチェックの総合ステータスを表す値。
const (
	// Degraded は、一部の依存が不調だが稼働継続できる状態を表します。
	Degraded = "degraded"
	// Ok は、すべて正常な状態を表します。
	Ok = "ok"
	// Unhealthy は、サービスが正常に応答できない状態を表します。
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
	clock         clock.Clock
	dbSystemQuery query.DBSystemQuery
}

// Usecase は、システムの健全性チェックに関するユースケースを定義します。
type Usecase interface {
	// CheckHealth は健全性 DTO を返します。異常時は nil を返し、DTO は参照しないこと。
	CheckHealth(ctx context.Context) (*DTO, error)
}

// New は、システムの健全性チェックに関するユースケースを初期化します。
func New(dbsq query.DBSystemQuery, tf observability.TracerFactory, clock clock.Clock) Usecase {
	return &usecase{
		tracer:        tf.Usecase(),
		clock:         clock,
		dbSystemQuery: dbsq,
	}
}

// CheckHealth は、システムの健全性をチェックするユースケースです。
func (u *usecase) CheckHealth(ctx context.Context) (*DTO, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	applTime := u.clock.Now()
	dbHealth, err := u.dbSystemQuery.CheckDBHealth(ctx)
	if err != nil {
		return nil, err
	}

	return &DTO{
		Status:          Ok,
		ApplicationTime: applTime,
		DBHealthCheck:   dbHealth,
	}, nil
}
