//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package address は、外部の郵便番号 lookup サービスへの意味的 gateway を提供します。
package address

import "context"

// Gateway は、外部サービスから郵便番号に対応する住所候補を取得する意味的 gateway です。
type Gateway interface {
	// Lookup は、正規化済み 7 桁の郵便番号に対応する住所候補を取得します。
	// 該当が無い場合は空スライスと nil error を返します。外部サービスの障害・不正応答時は
	// apperror sentinel（ErrUnavailable 等）を返します（外部レスポンスの型・エラーは gateway 内で完結）。
	Lookup(ctx context.Context, postalCode string) ([]*Candidate, error)
}

// Candidate は、住所候補 1 件を表す出力 DTO です。都道府県名は外部 lookup が返すフル表記
// （例 "東京都"）であり、prefecture_id への解決は usecase 層が担います。
type Candidate struct {
	// PrefectureName は、都道府県名（フル表記）です。
	PrefectureName string
	// City は、市区町村です。
	City string
	// Town は、町域です。
	Town string
}
