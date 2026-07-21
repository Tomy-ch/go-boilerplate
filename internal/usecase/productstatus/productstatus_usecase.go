//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package productstatus は、商品ステータスマスタの参照ユースケースを提供します。
package productstatus

import (
	"context"

	"go-boilerplate/internal/domain/productstatus"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

// ProductStatusDTO は、商品ステータス 1 件分のユースケース出力 DTO です。
type ProductStatusDTO struct {
	ID      uuid.UUID
	Code    int
	Name    string
	SortKey int
}

// ProductStatusDTOs は、ProductStatusDTO の一覧（sortKey 昇順）です。
type ProductStatusDTOs []ProductStatusDTO

// Usecase は、商品ステータスマスタの参照ユースケースを定義します。
type Usecase interface {
	// ListProductStatuses は、全商品ステータスを sortKey 昇順で返します。
	ListProductStatuses(ctx context.Context) (ProductStatusDTOs, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	repo   productstatus.Repository
}

// New は、商品ステータスマスタの参照ユースケースを生成します。
func New(repo productstatus.Repository, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		repo:   repo,
	}
}

// ListProductStatuses は、全商品ステータスを sortKey 昇順で返します。
func (u *usecase) ListProductStatuses(ctx context.Context) (ProductStatusDTOs, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	productStatuses, err := u.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make(ProductStatusDTOs, len(productStatuses))
	for i, p := range productStatuses {
		dtos[i] = ProductStatusDTO{
			ID:      p.ID(),
			Code:    p.Code(),
			Name:    p.Name(),
			SortKey: p.SortKey(),
		}
	}

	return dtos, nil
}
