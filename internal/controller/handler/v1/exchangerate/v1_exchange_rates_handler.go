//go:generate oapi-codegen --include-tags=v1/exchange-rates --package=gen --generate=types -o ./gen/type.gen.go /app/openapi/openapi.gen.yaml
//go:generate oapi-codegen --include-tags=v1/exchange-rates --package=gen --generate=echo5-server,strict-server -o ./gen/server.gen.go /app/openapi/openapi.gen.yaml

// Package exchangerate は、/v1/exchange-rates エンドポイントに関連するハンドラを提供します。
package exchangerate

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/handler/v1/exchangerate/gen"
	"go-boilerplate/internal/observability"
	exchangerateuc "go-boilerplate/internal/usecase/exchangerate"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v5"
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

	var displayCurrency *string
	if request.Params.DisplayCurrency != nil {
		dc := string(*request.Params.DisplayCurrency)
		displayCurrency = &dc
	}

	amount, err := decimal.Parse(request.Params.Amount)
	if err != nil {
		return nil, xerrors.Wrap(apperror.ErrInvalidArgument, "amount must be a decimal string")
	}

	result, err := s.uc.Convert(ctx, exchangerateuc.ConvertInput{
		Base:            request.Params.Base,
		Quote:           request.Params.Quote,
		Amount:          amount,
		DisplayCurrency: displayCurrency,
	})
	if err != nil {
		return nil, err
	}

	return gen.GetExchangeRates200JSONResponse(gen.ExchangeRateResponse{
		Base:            request.Params.Base,
		Quote:           request.Params.Quote,
		Amount:          amount.String(),
		Converted:       result.Converted.String(),
		ReferenceAmount: toReferenceAmount(result.Reference),
	}), nil
}

// toReferenceAmount は、usecase の参考換算額を API レスポンス DTO へ変換します。degrade 時は nil です。
func toReferenceAmount(r *exchangerateuc.ReferenceAmount) *gen.ReferenceAmount {
	if r == nil {
		return nil
	}
	return &gen.ReferenceAmount{
		Currency: r.Currency,
		Amount:   r.Amount,
		Rate:     r.Rate.String(),
		RateDate: r.RateDate,
	}
}
