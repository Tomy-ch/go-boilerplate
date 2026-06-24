//go:generate mockgen -source=$GOFILE -destination=mock/mock_exchangerate_usecase.gen.go -package=mock_$GOPACKAGE

// Package exchangerate は、為替レート gateway を用いた換算ユースケースのサンプルです。
package exchangerate

import (
	"context"

	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
)

// Usecase は、為替換算ユースケースを表します。
type Usecase interface {
	// Convert は、amount（base 通貨建て）を quote 通貨へ換算した値を返します。
	Convert(ctx context.Context, base, quote string, amount float64) (float64, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	gateway boundary.Gateway
	tracer  observability.LayerTracer
}

// New は、為替換算ユースケースを生成します。
func New(gateway boundary.Gateway, tf observability.TracerFactory) Usecase {
	return &usecase{
		gateway: gateway,
		tracer:  tf.Usecase(),
	}
}

// Convert は、gateway から取得したレートで amount を換算します。
func (u *usecase) Convert(ctx context.Context, base, quote string, amount float64) (float64, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	rate, err := u.gateway.GetRate(ctx, base, quote)
	if err != nil {
		return 0, err
	}
	return amount * rate.Value, nil
}
