//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package exchangerate は、為替レート gateway を用いた換算ユースケースのサンプルです。
package exchangerate

import (
	"context"
	"math"

	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
	"go-boilerplate/internal/usecase/tools/money"
)

// baseMinorUnitScale は、参考換算の入力 amount をセント精度（小数 2 桁）へ量子化するスケールです。
// amount×100 で整数化し ApplyRateHalfUp の scale へ渡すため乗除で相殺し、換算結果の桁は base 通貨の
// 小数桁に依存しません（amount の小数 3 桁以降のみ丸め込まれます）。入力正規化と丸めを 1 点に集約します。
const baseMinorUnitScale = 100

// Usecase は、為替換算ユースケースを表します。
type Usecase interface {
	// Convert は、base 通貨建ての金額を quote 通貨へ換算し、任意で表示通貨の参考換算額を返します。
	Convert(ctx context.Context, in ConvertInput) (*ConvertResult, error)
}

// ConvertInput は、換算要求の入力 DTO です。
type ConvertInput struct {
	// Base は、換算元の通貨コードです。
	Base string
	// Quote は、換算先の通貨コードです。
	Quote string
	// Amount は、換算する金額（base 通貨建て）です。
	Amount float64
	// DisplayCurrency は、参考換算額の表示通貨です。未指定なら参考換算額を返しません。
	DisplayCurrency *string
}

// ConvertResult は、換算結果の出力 DTO です。
type ConvertResult struct {
	// Converted は、Quote 通貨へ換算した金額です。
	Converted float64
	// Reference は、表示通貨での参考換算額です。未指定時および degrade 時は nil です。
	Reference *ReferenceAmount
}

// ReferenceAmount は、表示通貨での参考換算額です（非永続・参考表示専用）。
type ReferenceAmount struct {
	// Currency は、参考換算額の通貨コードです。
	Currency string
	// Amount は、丸め後の参考換算額（表示通貨の最小単位整数）です。
	Amount int64
	// Rate は、換算に用いたレートです。
	Rate float64
	// RateDate は、レートの基準日です。
	RateDate string
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

// Convert は、base 通貨建ての金額を quote 通貨へ換算し、任意で表示通貨の参考換算額を返します。
func (u *usecase) Convert(ctx context.Context, in ConvertInput) (*ConvertResult, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	rate, err := u.gateway.GetRate(ctx, in.Base, in.Quote)
	if err != nil {
		return nil, err
	}

	result := &ConvertResult{Converted: in.Amount * rate.Value}

	// 参考レートの取得に失敗しても degrade（Reference = nil）とし、本体換算は継続させる。
	if in.DisplayCurrency != nil {
		if refRate, refErr := u.gateway.GetRate(ctx, in.Base, *in.DisplayCurrency); refErr == nil {
			amountMinor := int64(math.Round(in.Amount * baseMinorUnitScale))
			result.Reference = BuildReferenceAmount(refRate, amountMinor)
		}
	}
	return result, nil
}

// BuildReferenceAmount は、レートと base 最小単位額から参考換算額を組み立てます。
// reference_amount の組み立てを 1 箇所へ集約し、後続 API（purchases 等）から再利用します。
func BuildReferenceAmount(rate *boundary.Rate, amountMinor int64) *ReferenceAmount {
	return &ReferenceAmount{
		Currency: rate.Quote,
		Amount:   money.ApplyRateHalfUp(amountMinor, rate.Value, baseMinorUnitScale),
		Rate:     rate.Value,
		RateDate: rate.Date,
	}
}
