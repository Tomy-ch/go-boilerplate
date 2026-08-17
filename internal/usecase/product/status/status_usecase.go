//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package status は、商品ステータスマスタの参照ユースケースを提供します。
package status

import (
	"context"

	"go-boilerplate/internal/domain/product/status"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

// StatusDTO は、商品ステータス 1 件分のユースケース出力 DTO です。
type StatusDTO struct {
	ID   uuid.UUID
	Code int
	Name string
}

// StatusDTOs は、StatusDTO の一覧です。並びはマスタの表示順で、順序そのものが表示順を表します。
type StatusDTOs []StatusDTO

// Usecase は、商品ステータスマスタの参照ユースケースを定義します。
type Usecase interface {
	// ListStatuses は、全商品ステータスをマスタの表示順で返します。表示順の値は外へ出しません。
	ListStatuses(ctx context.Context) (StatusDTOs, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	repo   status.Repository
}

// New は、商品ステータスマスタの参照ユースケースを生成します。
func New(repo status.Repository, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		repo:   repo,
	}
}

func (u *usecase) ListStatuses(ctx context.Context) (StatusDTOs, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	statuses, err := u.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make(StatusDTOs, len(statuses))
	for i, p := range statuses {
		dtos[i] = StatusDTO{
			ID:   p.ID(),
			Code: p.Code(),
			Name: p.Name(),
		}
	}

	return dtos, nil
}
