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
// ProductName は購入時点ではなく現在の商品名、UnitPrice は価格スケール（ドル decimal）です。
type PurchaseDetailItemView struct {
	ProductID   uuid.UUID
	ProductName string
	Quantity    int
	UnitPrice   decimal.Decimal
}

// PurchaseGetDetailView は、購入詳細（取得）1 件分のユースケース出力 DTO です。金額はすべて USD セント単位の整数、
// ステータスは購入ステータスマスタで解決済みの ID と名称、PaidAt / CanceledAt は未確定なら nil です。
type PurchaseGetDetailView struct {
	Code           string
	UserID         uuid.UUID
	StatusID       uuid.UUID
	StatusCode     int
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

// GetPurchaseDetail は、購入と商品にまたがる read 投影を QueryService に委ねます。所有権の絞り込みも
// QueryService 側が担うため、usecase 側では取得後の所有者チェックを行いません。
func (u *usecase) GetPurchaseDetail(ctx context.Context, authn *auth.Authn, purchaseCode string) (PurchaseGetDetailView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return PurchaseGetDetailView{}, xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")
	}
	userID, err := authn.UserID()
	if err != nil {
		return PurchaseGetDetailView{}, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	rm, err := u.detailQS.FindDetailByUserAndCode(ctx, userID, purchaseCode)
	if err != nil {
		return PurchaseGetDetailView{}, err
	}

	return toPurchaseGetDetailView(rm), nil
}

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
		Code:           rm.Code,
		UserID:         rm.UserID,
		StatusID:       rm.StatusID,
		StatusCode:     rm.StatusCode,
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
