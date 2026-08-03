package product

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
)

const (
	// defaultLimit は、limit 未指定時に用いる既定の取得件数です。
	defaultLimit = 20
	// minLimit / maxLimit は、取得件数のクランプ範囲です。
	minLimit = 1
	maxLimit = 100
)

// ListLowStockProductsParams は、在庫僅少商品一覧取得の入力パラメータです。
type ListLowStockProductsParams struct {
	// Limit は、取得する上位件数です。1 未満は既定値 20 を適用し、100 を超える値は 100 にクランプします。
	Limit int
}

// ProductLowStockListView は、在庫僅少商品一覧の取得結果を表します。
type ProductLowStockListView struct {
	// Items は、在庫の少ない順（同数は商品 ID 昇順）で並んだ在庫僅少商品の一覧です。
	Items []ProductView
}

// ListLowStockProducts は、admin 認可のうえ、在庫が在庫警告閾値以下まで減った商品を在庫の少ない順に返します。
func (u *usecase) ListLowStockProducts(
	ctx context.Context, authn *auth.Authn, params ListLowStockProductsParams,
) (ProductLowStockListView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return ProductLowStockListView{}, apperror.ErrUnauthenticated
	}
	if err := u.authorizer.Authorize(
		ctx, authn, authz.ActionProductListLowStock, authz.NewResource("product", nil),
	); err != nil {
		return ProductLowStockListView{}, err
	}

	//nolint:gosec // G115: normalizeLimit が [minLimit, maxLimit] にクランプ済みのため範囲に収まります
	products, err := u.repo.FindAllLowStock(ctx, int32(normalizeLimit(params.Limit)))
	if err != nil {
		return ProductLowStockListView{}, err
	}

	items := make([]ProductView, len(products))
	for i, p := range products {
		items[i] = toProductView(p)
	}

	return ProductLowStockListView{Items: items}, nil
}

// normalizeLimit は、取得件数を既定値の適用とクランプで正規化します。
func normalizeLimit(limit int) int {
	if limit < minLimit {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
