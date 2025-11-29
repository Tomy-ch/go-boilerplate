//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE -package=mock_$GOPACKAGE

// Package healthcheckuc は、システムの健全性チェックに関するユースケースを提供します。
package healthcheckuc

import (
	"context"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/usecase/healthcheck/query"
)

// DTO は、システムの健全性に関するデータ転送用のオブジェクトです。
type DTO struct {
	Status          string
	ApplicationTime time.Time
	DBHealthCheck   query.DBHealth
}

// usecase は、システムの健全性チェックに関するユースケースを提供します。
type usecase struct {
	cfg           *config.Config
	dbSystemQuery query.DBSystemQuery
}

// Usecase は、システムの健全性チェックに関するユースケースを定義します。
type Usecase interface {
	CheckHealth(ctx context.Context) (DTO, error)
}

const (
	Degraded  = "degraded"
	Ok        = "ok"
	Unhealthy = "unhealthy"
)

// New は、システムの健全性チェックに関するユースケースを初期化します。
func New(dbsq query.DBSystemQuery, cfg *config.Config) Usecase {
	return &usecase{
		dbSystemQuery: dbsq,
		cfg:           cfg,
	}
}

// CheckHealth は、システムの健全性をチェックするユースケースです。
func (u *usecase) CheckHealth(ctx context.Context) (DTO, error) {
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
