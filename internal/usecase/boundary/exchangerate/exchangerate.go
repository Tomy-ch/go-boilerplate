//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package exchangerate は、外部為替レート取得サービスへの意味的 gateway を提供します。
package exchangerate

import (
	"context"

	"go-boilerplate/pkg/decimal"
)

// Gateway は、外部サービスから為替レートを取得する意味的 gateway です。
type Gateway interface {
	// GetRate は、base 通貨を quote 通貨へ換算するレートを取得します。
	// 失敗時は apperror sentinel（ErrUnavailable / ErrNotFound 等）を返します。
	GetRate(ctx context.Context, base, quote string) (*Rate, error)
}

// Rate は、為替レートの取得結果を表す出力 DTO です。
type Rate struct {
	// Base は、換算元の通貨コードです。
	Base string
	// Quote は、換算先の通貨コードです。
	Quote string
	// Value は、Base 1 単位あたりの Quote 換算値です。源泉は正確な十進量であり、float は取込時点で値を破壊するため Decimal で保持します。
	Value decimal.Decimal
	// Date は、レートの基準日（外部レートサービスの公表日、例 "2026-07-21"）です。
	// 外部レスポンスに date が含まれない場合は空文字になります。
	Date string
}
