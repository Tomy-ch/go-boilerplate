package event

import (
	"encoding/json"
	"time"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/pkg/xerrors"
)

// TypeCanceled は、購入キャンセルの outbox イベント種別（version 込み）です。
const TypeCanceled = "purchase.canceled.v1"

// canceled は、purchase.canceled.v1 の自己完結 snapshot payload です。
type canceled struct {
	PurchaseID string `json:"purchaseId"`
	Code       string `json:"code"`
	UserID     string `json:"userId"`
	StatusCode int    `json:"statusCode"`
	CanceledAt string `json:"canceledAt"`
}

// BuildCanceled は、購入集約から purchase.canceled.v1 の自己完結 snapshot payload を marshal します。
func BuildCanceled(p *purchase.Purchase) ([]byte, error) {
	var canceledAt string
	if at := p.CanceledAt(); at != nil {
		canceledAt = at.Format(time.RFC3339Nano)
	}

	payload, err := json.Marshal(canceled{
		PurchaseID: p.ID().String(),
		Code:       p.Code(),
		UserID:     p.UserID().String(),
		StatusCode: p.StatusCode(),
		CanceledAt: canceledAt,
	})
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to encode purchase.canceled payload")
	}
	return payload, nil
}
