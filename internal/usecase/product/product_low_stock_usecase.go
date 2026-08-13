package product

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/xerrors"
)

const (
	// lowStockDefaultLimit は、limit 未指定時に用いる既定の取得件数です。
	lowStockDefaultLimit = 20
	// lowStockMaxLimit は、許容する最大の取得件数です。
	lowStockMaxLimit = 100
)

// lowStockLimitPolicy は、在庫僅少商品一覧の取得件数規約です。OpenAPI の limit（既定 20 / 1〜100）と対応します。
var (
	lowStockLimitPolicy = paging.LimitPolicy{Default: lowStockDefaultLimit, Max: lowStockMaxLimit}

	// errNotLowStockInLowStockRead は、在庫僅少として取得した読み取りに該当しない商品が混じっていた場合の
	// エラーです。絞り込みを実行する SQL と、在庫僅少を定義する Product.IsLowStock が食い違ったことを
	// 意味します。
	errNotLowStockInLowStockRead = xerrors.Wrap(apperror.ErrInternal, "product not low on stock in low-stock read")
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

	limit := paging.NewLimit(ptr.To(params.Limit), lowStockLimitPolicy)
	products, err := u.repo.FindAllLowStock(ctx, limit.Value32())
	if err != nil {
		return ProductLowStockListView{}, err
	}
	// SQL と Product.IsLowStock の乖離を検出する（README の Verifying infrastructure against the domain）。
	for _, p := range products {
		if !p.IsLowStock() {
			return ProductLowStockListView{}, xerrors.Wrap(errNotLowStockInLowStockRead, p.ID().String())
		}
	}

	items := make([]ProductView, len(products))
	for i, p := range products {
		items[i] = toProductView(p)
	}

	return ProductLowStockListView{Items: items}, nil
}
