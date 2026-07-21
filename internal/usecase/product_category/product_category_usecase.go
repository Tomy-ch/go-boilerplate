//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package productcategory は、商品カテゴリマスタの参照ユースケースを提供します。
package productcategory

import (
	"context"

	productcategory "go-boilerplate/internal/domain/product_category"
	"go-boilerplate/internal/observability"
	"go-boilerplate/pkg/uuid"
)

// ProductCategoryDTO は、商品カテゴリ 1 件分のユースケース出力 DTO です。
type ProductCategoryDTO struct {
	ID      uuid.UUID
	Code    int
	Name    string
	SortKey int
}

// ProductCategoryDTOs は、ProductCategoryDTO の一覧（sortKey 昇順）です。
type ProductCategoryDTOs []ProductCategoryDTO

// Usecase は、商品カテゴリマスタの参照ユースケースを定義します。
type Usecase interface {
	// ListProductCategories は、全商品カテゴリを sortKey 昇順で返します。
	ListProductCategories(ctx context.Context) (ProductCategoryDTOs, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	repo   productcategory.Repository
}

// New は、商品カテゴリマスタの参照ユースケースを生成します。
func New(repo productcategory.Repository, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		repo:   repo,
	}
}

// ListProductCategories は、全商品カテゴリを sortKey 昇順で返します。
func (u *usecase) ListProductCategories(ctx context.Context) (ProductCategoryDTOs, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	categories, err := u.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make(ProductCategoryDTOs, len(categories))
	for i, c := range categories {
		dtos[i] = ProductCategoryDTO{
			ID:      c.ID(),
			Code:    c.Code(),
			Name:    c.Name(),
			SortKey: c.SortKey(),
		}
	}

	return dtos, nil
}
