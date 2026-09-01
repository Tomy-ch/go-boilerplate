//go:generate oapi-codegen --include-tags=v1/purchases/statuses --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/purchases/statuses --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package purchasestatuses は、/v1/purchases/statuses エンドポイントに関連するハンドラを提供します。
package purchasestatuses

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/purchases/statuses/gen"
	"go-boilerplate/internal/observability"
	statusuc "go-boilerplate/internal/usecase/purchase/status"

	"github.com/labstack/echo/v5"
)

type server struct {
	tracer observability.LayerTracer
	uc     statusuc.Usecase
}

// BindHandler は、購入ステータスマスタ一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc statusuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetPurchaseStatuses は、購入ステータスマスタの全件をマスタの表示順で返します。表示順の値は応答に含めません。
func (s *server) GetPurchaseStatuses(
	ctx context.Context, _ gen.GetPurchaseStatusesRequestObject,
) (gen.GetPurchaseStatusesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	list, err := s.uc.ListStatuses(ctx)
	if err != nil {
		return nil, err
	}

	purchaseStatuses := make([]gen.PurchaseStatusRef, len(list))
	for i, dto := range list {
		purchaseStatuses[i] = toPurchaseStatusRef(dto)
	}

	return gen.GetPurchaseStatuses200JSONResponse(purchaseStatuses), nil
}

// toPurchaseStatusRef は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toPurchaseStatusRef(dto statusuc.StatusDTO) gen.PurchaseStatusRef {
	return gen.PurchaseStatusRef{
		Id:   dto.ID.ToPrimitive(),
		Code: int64(dto.Code),
		Name: dto.Name,
	}
}
