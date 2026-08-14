//go:generate oapi-codegen --include-tags=v1/carts/items --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/carts/items --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package items は、/v1/carts/me/items エンドポイントに関連するハンドラを提供します。
package items

import (
	"context"

	"go-boilerplate/internal/controller/conv"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/carts/items/gen"
	"go-boilerplate/internal/observability"
	cartuc "go-boilerplate/internal/usecase/cart"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     cartuc.Usecase
}

// BindHandler は、カート明細のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc cartuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// PutCartsMeItem は、呼び出し主体のカートに指定商品の数量を設定します。
func (s *server) PutCartsMeItem(
	ctx context.Context, request gen.PutCartsMeItemRequestObject,
) (gen.PutCartsMeItemResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	subject, err := toSubject(ctx, request.Params)
	if err != nil {
		return nil, err
	}

	view, err := s.uc.SetItem(ctx, cartuc.SetItemParams{
		Subject:   subject,
		ProductID: conv.UUID(request.ProductId),
		Quantity:  int(request.Body.Quantity),
	})
	if err != nil {
		return nil, err
	}

	response, err := toCartResponse(view)
	if err != nil {
		return nil, err
	}

	return gen.PutCartsMeItem200JSONResponse(response), nil
}

// toSubject は、認証結果とヘッダからカートの主体を組み立てます。
// 認証はこの operation では任意で、認証済みの呼び出し元はゲストセッションより優先されます。
// どちらも無い呼び出し元は主体を持ちません（それ以降の振る舞いは usecase が持ちます）。
func toSubject(ctx context.Context, params gen.PutCartsMeItemParams) (cartuc.Subject, error) {
	if authn, ok := ctxhelper.GetAuthn(ctx); ok {
		userID, err := authn.UserID()
		if err != nil {
			return cartuc.Subject{}, xerrors.Wrap(err, "failed to get user ID from authenticator")
		}
		return cartuc.Subject{UserID: &userID}, nil
	}
	return cartuc.Subject{SessionToken: params.XCartSession}, nil
}

// toCartResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toCartResponse(view cartuc.CartView) (gen.CartResponse, error) {
	items, err := toCartItemResponses(view.Items)
	if err != nil {
		return gen.CartResponse{}, err
	}

	return gen.CartResponse{
		SessionToken:   view.SessionToken,
		Items:          items,
		SubtotalAmount: view.SubtotalAmount,
		ExpiresAt:      view.ExpiresAt,
	}, nil
}

// toCartItemResponses は、明細の DTO を HTTP レスポンスへ変換します。
func toCartItemResponses(views []cartuc.CartItemView) ([]gen.CartItemResponse, error) {
	items := make([]gen.CartItemResponse, len(views))
	for i, v := range views {
		quantity, err := safecast.IntToInt32(v.Quantity)
		if err != nil {
			return nil, xerrors.Wrap(err, "invalid cart item quantity")
		}

		available, err := toAvailableQuantity(v.AvailableQuantity)
		if err != nil {
			return nil, err
		}

		items[i] = gen.CartItemResponse{
			ProductId:         v.ProductID.ToPrimitive(),
			ProductName:       v.ProductName,
			Quantity:          quantity,
			UnitPrice:         toUnitPrice(v.UnitPrice),
			Issues:            toCartItemIssues(v.Issues),
			AvailableQuantity: available,
		}
	}
	return items, nil
}

// toAvailableQuantity は、在庫不足時の購入可能上限を変換します。指定が無い場合は nil を返します。
//
//nolint:nilnil // 上限が無いこと自体が正常な状態で、レスポンスでは null として表現される
func toAvailableQuantity(quantity *int) (*int32, error) {
	if quantity == nil {
		return nil, nil
	}
	converted, err := safecast.IntToInt32(*quantity)
	if err != nil {
		return nil, xerrors.Wrap(err, "invalid available quantity")
	}
	return &converted, nil
}

// toUnitPrice は、単価を decimal 文字列へ変換します。単価が無い場合は nil を返します。
func toUnitPrice(price *decimal.Decimal) *string {
	if price == nil {
		return nil
	}
	s := price.String()
	return &s
}

// toCartItemIssues は、再評価結果を HTTP レスポンスの列挙値へ変換します。
func toCartItemIssues(issues []cartuc.ItemIssue) []gen.CartItemIssue {
	converted := make([]gen.CartItemIssue, len(issues))
	for i, issue := range issues {
		converted[i] = gen.CartItemIssue(issue)
	}
	return converted
}
