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
	ID   uuid.UUID
	Code int
	Name string
}

// CategoryDTOs は、CategoryDTO の一覧です。並びはマスタの表示順で、順序そのものが表示順を表します。
type CategoryDTOs []CategoryDTO

// Usecase は、商品カテゴリマスタの参照ユースケースを定義します。
type Usecase interface {
	// ListCategories は、全商品カテゴリをマスタの表示順で返します。表示順の値は外へ出しません。
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
			ID:   c.ID(),
			Code: c.Code(),
			Name: c.Name(),
		}
	}

	return dtos, nil
}
