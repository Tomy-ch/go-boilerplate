//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package category は、商品カテゴリマスタの参照ユースケースを提供します。
package category

import (
	"context"

	"go-boilerplate/internal/domain/product/category"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

// CategoryDTO は、商品カテゴリ 1 件分のユースケース出力 DTO です。
type CategoryDTO struct {
	ID      uuid.UUID
	Code    int
	Name    string
	SortKey int
}

// CategoryDTOs は、CategoryDTO の一覧（sortKey 昇順）です。
type CategoryDTOs []CategoryDTO

// Usecase は、商品カテゴリマスタの参照ユースケースを定義します。
type Usecase interface {
	// ListCategories は、全商品カテゴリを sortKey 昇順で返します。
	ListCategories(ctx context.Context) (CategoryDTOs, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	repo   category.Repository
}

// New は、商品カテゴリマスタの参照ユースケースを生成します。
func New(repo category.Repository, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		repo:   repo,
	}
}

// ListCategories は、全商品カテゴリを sortKey 昇順で返します。
func (u *usecase) ListCategories(ctx context.Context) (CategoryDTOs, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	categories, err := u.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make(CategoryDTOs, len(categories))
	for i, c := range categories {
		dtos[i] = CategoryDTO{
			ID:      c.ID(),
			Code:    c.Code(),
			Name:    c.Name(),
			SortKey: c.SortKey(),
		}
	}

	return dtos, nil
}
