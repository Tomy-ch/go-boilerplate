//go:generate mockgen -source=$GOFILE -destination=mock/mock_exchangerate.gen.go -package=mock_$GOPACKAGE

// Package exchangerate は、外部為替レート取得サービスへの意味的 gateway を提供します（DTO モードのサンプル）。
package exchangerate

import "context"

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
	// Value は、Base 1 単位あたりの Quote 換算値です。
	Value float64
}
