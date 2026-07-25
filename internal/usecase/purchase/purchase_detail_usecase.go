package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// PurchaseDetailItemView は、購入詳細（取得）の明細 1 件のユースケース出力 DTO です。
// ProductName は products との結合で解決した現在名、UnitPrice は価格スケール（ドル decimal）です。
type PurchaseDetailItemView struct {
	ProductID   uuid.UUID
	ProductName string
	Quantity    int
	UnitPrice   decimal.Decimal
}

// PurchaseGetDetailView は、購入詳細（取得）1 件分のユースケース出力 DTO です。金額はすべて USD セント単位の整数、
// ステータスは購入ステータスマスタで解決済みの ID と名称、PaidAt / CanceledAt は未確定なら nil です。
type PurchaseGetDetailView struct {
	ID             uuid.UUID
	Code           string
	UserID         uuid.UUID
	StatusID       uuid.UUID
	StatusName     string
	SubtotalAmount int64
	TaxAmount      int64
	ShippingFee    int64
	TotalAmount    int64
	Details        []PurchaseDetailItemView
	OrderedAt      time.Time
	PaidAt         *time.Time
	CanceledAt     *time.Time
}

// GetPurchaseDetail は、本人の購入 1 件を明細（商品名込み）とともに取得します。
func (u *usecase) GetPurchaseDetail(ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID) (PurchaseGetDetailView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return PurchaseGetDetailView{}, xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")
	}
	userID, err := authn.UserID()
	if err != nil {
		return PurchaseGetDetailView{}, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	rm, err := u.detailQS.FindDetailByUserAndID(ctx, userID, purchaseID)
	if err != nil {
		return PurchaseGetDetailView{}, err
	}

	return toPurchaseGetDetailView(rm), nil
}

// toPurchaseGetDetailView は、購入詳細の読み取りモデルを取得レスポンスの出力 DTO へ変換します。
func toPurchaseGetDetailView(rm *query.PurchaseDetailReadModel) PurchaseGetDetailView {
	items := make([]PurchaseDetailItemView, len(rm.Items))
	for i, it := range rm.Items {
		items[i] = PurchaseDetailItemView{
			ProductID:   it.ProductID,
			ProductName: it.ProductName,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice.Decimal(),
		}
	}
	return PurchaseGetDetailView{
		ID:             rm.ID,
		Code:           rm.Code,
		UserID:         rm.UserID,
		StatusID:       rm.StatusID,
		StatusName:     rm.StatusName,
		SubtotalAmount: rm.SubtotalAmount,
		TaxAmount:      rm.TaxAmount,
		ShippingFee:    rm.ShippingFee,
		TotalAmount:    rm.TotalAmount,
		Details:        items,
		OrderedAt:      rm.OrderedAt,
		PaidAt:         rm.PaidAt,
		CanceledAt:     rm.CanceledAt,
	}
}
