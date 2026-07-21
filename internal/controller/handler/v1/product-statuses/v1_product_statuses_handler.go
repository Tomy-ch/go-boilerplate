//go:generate oapi-codegen --include-tags=v1/product-statuses --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/product-statuses --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package productstatuses は、/v1/product-statuses エンドポイントに関連するハンドラを提供します。
package productstatuses

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/product-statuses/gen"
	"go-boilerplate/internal/observability"
	productstatusuc "go-boilerplate/internal/usecase/productstatus"

	"github.com/labstack/echo/v4"
)

type server struct {
	tracer observability.LayerTracer
	uc     productstatusuc.Usecase
}

// BindHandler は、商品ステータスマスタ一覧のハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc productstatusuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetProductStatuses は、商品ステータスマスタの全件を sort_key 昇順で返します。
func (s *server) GetProductStatuses(
	ctx context.Context, _ gen.GetProductStatusesRequestObject,
) (gen.GetProductStatusesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	list, err := s.uc.ListProductStatuses(ctx)
	if err != nil {
		return nil, err
	}

	productStatuses := make([]gen.ProductStatusResponse, len(list))
	for i, dto := range list {
		productStatuses[i] = toProductStatusResponse(dto)
	}

	return gen.GetProductStatuses200JSONResponse(productStatuses), nil
}

// toProductStatusResponse は、ユースケースの DTO を HTTP レスポンスへ変換します。
func toProductStatusResponse(dto productstatusuc.ProductStatusDTO) gen.ProductStatusResponse {
	return gen.ProductStatusResponse{
		Id:      dto.ID.ToPrimitive(),
		Code:    dto.Code,
		Name:    dto.Name,
		SortKey: dto.SortKey,
	}
}
