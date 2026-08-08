// Package event は、購入ユースケースが発行する outbox イベントの本文（自己完結 snapshot）と
// その marshal を提供します。版付きのイベント種別と JSON のワイヤ表現を本パッケージへ隔離し、
// usecase 本体を薄く保ちます（ADR-0047）。
package event

import (
	"encoding/json"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/pkg/xerrors"
)

// TypeCreated は、購入作成の outbox イベント種別（version 込み）です。
const TypeCreated = "purchase.created.v1"

// created は、purchase.created.v1 の自己完結 snapshot payload です。金額は決済スケール（整数セント）です。
type created struct {
	PurchaseID     string          `json:"purchaseId"`
	Code           string          `json:"code"`
	UserID         string          `json:"userId"`
	StatusCode     int             `json:"statusCode"`
	SubtotalAmount int             `json:"subtotalAmount"`
	TaxAmount      int             `json:"taxAmount"`
	ShippingFee    int             `json:"shippingFee"`
	TotalAmount    int             `json:"totalAmount"`
	Details        []createdDetail `json:"details"`
}

// createdDetail は、購入明細の snapshot です。UnitPrice は価格スケール（ドル decimal）を可逆に保つため文字列で保持します。
type createdDetail struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unitPrice"`
}

// BuildCreated は、購入集約から purchase.created.v1 の自己完結 snapshot payload を marshal します。
func BuildCreated(p *purchase.Purchase) ([]byte, error) {
	src := p.Details()
	details := make([]createdDetail, len(src))
	for i, d := range src {
		details[i] = createdDetail{
			ProductID: d.ProductID().String(),
			Quantity:  d.Quantity(),
			UnitPrice: d.UnitPrice().String(),
		}
	}

	payload, err := json.Marshal(created{
		PurchaseID:     p.ID().String(),
		Code:           p.Code(),
		UserID:         p.UserID().String(),
		StatusCode:     p.StatusCode(),
		SubtotalAmount: p.SubtotalAmount(),
		TaxAmount:      p.TaxAmount(),
		ShippingFee:    p.ShippingFee(),
		TotalAmount:    p.TotalAmount(),
		Details:        details,
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to encode purchase.created payload")
	}
	return payload, nil
}
