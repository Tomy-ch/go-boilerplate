//go:generate oapi-codegen --include-tags=v1/exchange-rates --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/exchange-rates --package=gen --generate=echo-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package exchangerate は、/v1/exchange-rates エンドポイントに関連するハンドラを提供します。
package exchangerate

import (
	"context"

	"go-boilerplate/internal/controller/handler/v1/exchangerate/gen"
	"go-boilerplate/internal/observability"
	exchangerateuc "go-boilerplate/internal/usecase/exchangerate"

	"github.com/labstack/echo/v4"
)

type server struct {
	tracer observability.LayerTracer
	uc     exchangerateuc.Usecase
}

// BindHandler は、為替換算エンドポイントのハンドラを Echo に登録します。
func BindHandler(e *echo.Echo, tf observability.TracerFactory, uc exchangerateuc.Usecase) {
	gen.RegisterHandlers(e, gen.NewStrictHandler(&server{
		tracer: tf.Controller(),
		uc:     uc,
	}, nil))
}

// GetExchangeRates は、為替レートで amount を換算した結果を返します。
func (s *server) GetExchangeRates(
	ctx context.Context, request gen.GetExchangeRatesRequestObject,
) (gen.GetExchangeRatesResponseObject, error) {
	ctx, endSpan := s.tracer.Start(ctx)
	defer endSpan()

	converted, err := s.uc.Convert(ctx, request.Params.Base, request.Params.Quote, request.Params.Amount)
	if err != nil {
		return nil, err
	}

	return gen.GetExchangeRates200JSONResponse(gen.ExchangeRateResponse{
		Base:      request.Params.Base,
		Quote:     request.Params.Quote,
		Amount:    request.Params.Amount,
		Converted: converted,
	}), nil
}
