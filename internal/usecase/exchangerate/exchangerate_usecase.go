//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package exchangerate は、為替レート gateway を用いた換算ユースケースのサンプルです。
package exchangerate

import (
	"context"

	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
	"go-boilerplate/internal/usecase/tools/money"
	"go-boilerplate/pkg/decimal"
)

// displayMinorUnitDigits は、参考換算額を表示通貨の最小単位へ丸める際の小数桁数です。
// 現状の表示通貨は JPY のみで最小単位は 1 円（小数 0 桁）です。丸めは決済境界のこの 1 点でのみ行います。
const displayMinorUnitDigits = 0

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
	// Amount は、換算する金額（base 通貨建て）です。正確な十進量として扱い float を経由しません。
	Amount decimal.Decimal
	// DisplayCurrency は、参考換算額の表示通貨です。未指定なら参考換算額を返しません。
	DisplayCurrency *string
}

// ConvertResult は、換算結果の出力 DTO です。
type ConvertResult struct {
	// Converted は、Quote 通貨へ換算した金額です。正確な十進量として保持します。
	Converted decimal.Decimal
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
	Rate decimal.Decimal
	// RateDate は、レートの基準日です。
	RateDate string
}

// usecase は、Usecase の実装です。
type usecase struct {
	gateway boundary.Gateway
	logging logging.Logger
	tracer  observability.LayerTracer
}

// New は、為替換算ユースケースを生成します。
func New(gateway boundary.Gateway, logger logging.Logger, tf observability.TracerFactory) Usecase {
	return &usecase{
		gateway: gateway,
		logging: logger,
		tracer:  tf.Usecase(),
	}
}

func (u *usecase) Convert(ctx context.Context, in ConvertInput) (*ConvertResult, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	rate, err := u.gateway.GetRate(ctx, in.Base, in.Quote)
	if err != nil {
		return nil, err
	}

	result := &ConvertResult{Converted: in.Amount.Mul(rate.Value)}

	// 参考換算額は degrade（Reference = nil）しても本体換算は継続させる。無音にせず理由を warn ログへ残す。
	if in.DisplayCurrency != nil {
		refRate, refErr := u.gateway.GetRate(ctx, in.Base, *in.DisplayCurrency)
		switch {
		case refErr != nil:
			u.logging.Warn(ctx, "reference amount degraded: display rate unavailable", logging.Error(logging.ErrorKey, refErr))
		default:
			ref, buildErr := BuildReferenceAmount(refRate, in.Amount)
			if buildErr != nil {
				u.logging.Warn(ctx, "reference amount degraded: amount exceeds settlement range", logging.Error(logging.ErrorKey, buildErr))
			} else {
				result.Reference = ref
			}
		}
	}
	return result, nil
}

// BuildReferenceAmount は、レートと base 通貨建て金額から参考換算額を組み立てます。
// 金額が表示通貨の最小単位整数の範囲を超える場合はエラーを返します。
func BuildReferenceAmount(rate *boundary.Rate, amount decimal.Decimal) (*ReferenceAmount, error) {
	minor, err := money.ApplyRateHalfUp(amount, rate.Value, displayMinorUnitDigits)
	if err != nil {
		return nil, err
	}
	return &ReferenceAmount{
		Currency: rate.Quote,
		Amount:   minor,
		Rate:     rate.Value,
		RateDate: rate.Date,
	}, nil
}
