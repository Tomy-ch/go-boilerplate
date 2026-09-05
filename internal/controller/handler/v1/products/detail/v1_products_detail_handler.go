//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/products/detail --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package detail は、/v1/products/{productId} エンドポイントに関連するハンドラを提供します。
package detail

import (
	"context"
	"net/http"
	"time"

	"go-boilerplate/internal/apperror"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/products/detail/gen"
	idempotencymw "go-boilerplate/internal/controller/httpstack/idempotency"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/idempotency"
	productuc "go-boilerplate/internal/usecase/product"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/patch"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
	"github.com/oapi-codegen/nullable"
)

type server struct {
	tracer observability.LayerTracer
	uc     productuc.Usecase
	idem   idempotency.Deps
}

// BindHandler は、商品詳細のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc productuc.Usecase, idem idempotency.Deps) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
		idem:   idem,
	}, []gen.StrictMiddlewareFunc{idempotencymw.StrictMiddleware[gen.StrictHandlerFunc]()}))
}

// GetProductsDetail は、指定された UUID に該当する商品の詳細情報を取得します。認証は任意です。
// 既定では未存在・非公開はいずれもユースケースが NotFound を返し、404 で存在を秘匿します。
// includeUnpublished の指定時のみ未公開も引けます。
func (s *server) GetProductsDetail(ctx context.Context, request gen.GetProductsDetailRequestObject) (gen.GetProductsDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	id := conv.UUID(request.ProductId)

	dto, err := s.uc.GetProduct(ctx, ctxhelper.OptionalAuthn(ctx), productuc.GetProductParams{
		ID:                 id,
		IncludeUnpublished: ptr.Deref(request.Params.IncludeUnpublished, false),
	})
	if err != nil {
		return nil, err
	}

	res, err := toProductResponse(dto)
	if err != nil {
		return nil, err
	}

	return gen.GetProductsDetail200JSONResponse(res), nil
}

// PatchProductsDetail は、指定された UUID に該当する商品の属性を部分更新します。
// 送信されたフィールドのみを更新し、null 指定はクリアとして扱います。
func (s *server) PatchProductsDetail(
	ctx context.Context,
	request gen.PatchProductsDetailRequestObject,
) (gen.PatchProductsDetailResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.ProductId)

	params := productuc.UpdateProductParams{
		Version:               int(request.Body.Version),
		Name:                  request.Body.Name,
		Price:                 request.Body.Price,
		Quantity:              ptr.Map(request.Body.Quantity, func(v int32) int { return int(v) }),
		CategoryID:            conv.UUIDPtr(request.Body.CategoryId),
		StatusID:              conv.UUIDPtr(request.Body.StatusId),
		Description:           toPatchField(request.Body.Description),
		StockWarningThreshold: toPatchFieldInt(request.Body.StockWarningThreshold),
		PublishedAt:           toPatchField(request.Body.PublishedAt),
		Images:                toPatchFieldImages(request.Body.Images),
	}

	dto, err := s.uc.UpdateProduct(ctx, &authn, id, params)
	if err != nil {
		return nil, err
	}

	res, err := toProductResponse(dto)
	if err != nil {
		return nil, err
	}

	return gen.PatchProductsDetail200JSONResponse(res), nil
}

// PatchProductsStock は、指定された UUID に該当する商品の在庫を増減します。
// delta は正で補充、負で差し引きを表し、増減後の在庫が負になる要求はユースケースが 422 で拒否します。
func (s *server) PatchProductsStock(
	ctx context.Context,
	request gen.PatchProductsStockRequestObject,
) (gen.PatchProductsStockResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	id := conv.UUID(request.ProductId)

	params := productuc.UpdateProductStockParams{Delta: int(request.Body.Delta)}

	dto, err := s.uc.UpdateProductStock(ctx, &authn, id, params)
	if err != nil {
		return nil, err
	}

	res, err := toProductResponse(dto)
	if err != nil {
		return nil, err
	}

	return gen.PatchProductsStock200JSONResponse(res), nil
}

// toPatchField は、リクエストの 3 状態フィールドを部分更新の指定状態へ変換します。
func toPatchField[T any](v nullable.Nullable[T]) patch.Field[T] {
	if !v.IsSpecified() {
		return patch.Unspecified[T]()
	}
	if v.IsNull() {
		return patch.Null[T]()
	}
	return patch.Value(v.MustGet())
}

// toPatchFieldInt は、リクエストの 3 状態 int32 フィールドをユースケースの DTO の int へ変換します。
func toPatchFieldInt(v nullable.Nullable[int32]) patch.Field[int] {
	if !v.IsSpecified() {
		return patch.Unspecified[int]()
	}
	if v.IsNull() {
		return patch.Null[int]()
	}
	return patch.Value(int(v.MustGet()))
}

// toPatchFieldImages は、リクエストの 3 状態の商品画像をユースケースの DTO へ変換します。
func toPatchFieldImages(v nullable.Nullable[[]gen.ProductImageInput]) patch.Field[[]productuc.ProductImageParams] {
	if !v.IsSpecified() {
		return patch.Unspecified[[]productuc.ProductImageParams]()
	}
	if v.IsNull() {
		return patch.Null[[]productuc.ProductImageParams]()
	}

	inputs := v.MustGet()
	params := make([]productuc.ProductImageParams, len(inputs))
	for i, in := range inputs {
		params[i] = productuc.ProductImageParams{ImagePath: in.ImagePath, DisplaySort: int(in.DisplaySort)}
	}

	return patch.Value(params)
}

// toProductImageItems は、商品画像の DTO を HTTP レスポンスへ変換します。
// 表示順が int32 に収まらない場合はエラーを返します。
func toProductImageItems(dtos []productuc.ProductImageItemView) ([]gen.ProductImageItem, error) {
	items := make([]gen.ProductImageItem, len(dtos))
	for i, dto := range dtos {
		displaySort, err := safecast.IntToInt32(dto.DisplaySort)
		if err != nil {
			return nil, xerrors.Wrap(err, "invalid product image display sort")
		}
		items[i] = gen.ProductImageItem{ImagePath: dto.Path, DisplaySort: displaySort}
	}
	return items, nil
}

// toProductResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
// 在庫数・在庫警告閾値・バージョンが int32 に収まらない場合はエラーを返します。
func toProductResponse(dto productuc.ProductView) (gen.ProductResponse, error) {
	quantity, err := safecast.IntToInt32(dto.Quantity)
	if err != nil {
		return gen.ProductResponse{}, xerrors.Wrap(err, "invalid product quantity")
	}

	threshold, err := safecast.IntPtrToInt32Ptr(dto.StockWarningThreshold)
	if err != nil {
		return gen.ProductResponse{}, xerrors.Wrap(err, "invalid product stock warning threshold")
	}

	version, err := safecast.IntToInt32(dto.Version)
	if err != nil {
		return gen.ProductResponse{}, xerrors.Wrap(err, "invalid product version")
	}

	images, err := toProductImageItems(dto.Images)
	if err != nil {
		return gen.ProductResponse{}, err
	}

	return gen.ProductResponse{
		Id:                    dto.ID.ToPrimitive(),
		Name:                  dto.Name,
		Description:           dto.Description,
		Price:                 dto.Price.String(),
		Quantity:              quantity,
		StockWarningThreshold: threshold,
		Status: gen.ProductStatusRef{
			Id:   dto.StatusID.ToPrimitive(),
			Name: dto.StatusName,
		},
		Category: gen.ProductCategoryRef{
			Id:   dto.CategoryID.ToPrimitive(),
			Name: dto.CategoryName,
		},
		PublishedAt:    dto.PublishedAt,
		DiscontinuedAt: dto.DiscontinuedAt,
		Images:         images,
		Version:        version,
	}, nil
}

// PostProductsDiscontinue は、指定された UUID に該当する商品を廃番にします。admin のみ実行できます。
// 進行中の購入が残っている場合はユースケースが Conflict を返し、409 になります。
func (s *server) PostProductsDiscontinue(
	ctx context.Context,
	request gen.PostProductsDiscontinueRequestObject,
) (gen.PostProductsDiscontinueResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	rate, err := decimal.Parse(request.Body.CouponDiscountRate)
	if err != nil {
		return nil, xerrors.Join(apperror.ErrInvalidArgument, err)
	}

	view, _, err := idempotency.Run(ctx, s.idem, http.StatusOK, func(ctx context.Context) (productuc.DiscontinueProductView, error) {
		return s.uc.DiscontinueProduct(ctx, &authn, conv.UUID(request.ProductId), productuc.DiscontinueProductParams{
			CouponDiscountRate: rate,
			CouponValidity:     time.Duration(request.Body.CouponValidityDays) * 24 * time.Hour,
		})
	})
	if err != nil {
		return nil, err
	}

	return gen.PostProductsDiscontinue200JSONResponse(gen.ProductDiscontinueResponse{
		DiscontinuedAt:    view.DiscontinuedAt,
		AffectedCartCount: view.AffectedCartCount,
		AffectedUserCount: view.AffectedUserCount,
		IssuedCouponCount: view.IssuedCouponCount,
	}), nil
}

// GetProductsDiscontinueImpact は、廃番にした場合の影響の見積もりを返します。admin のみ実行できます。
// 返す値の鮮度は openapi/paths/v1/products/productId/discontinue-impact.yaml の description を参照。
func (s *server) GetProductsDiscontinueImpact(
	ctx context.Context,
	request gen.GetProductsDiscontinueImpactRequestObject,
) (gen.GetProductsDiscontinueImpactResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	authn, err := ctxhelper.RequireAuthn(ctx)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.GetDiscontinueImpact(ctx, &authn, conv.UUID(request.ProductId))
	if err != nil {
		return nil, err
	}

	return gen.GetProductsDiscontinueImpact200JSONResponse(gen.ProductDiscontinueImpactResponse{
		AffectedCartCount:       view.AffectedCartCount,
		AffectedUserCount:       view.AffectedUserCount,
		InProgressPurchaseCount: view.InProgressPurchaseCount,
	}), nil
}
