//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package purchase は、購入の作成ユースケースを提供します。金額はすべて USD セント単位の整数です。
package purchase

import (
	"context"
	"encoding/json"
	"time"

	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/exchangerate"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/internal/usecase/purchase/command"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// baseCurrency は、購入金額の基軸通貨です。金額は本通貨のセント整数で保持します。
	baseCurrency = "USD"
	// eventTypeCreated は、購入作成の outbox イベント種別（version 込み）です。
	eventTypeCreated = "purchase.created.v1"
	// aggregateType は、outbox の集約種別です。
	aggregateType = "purchase"
	// centsPerBaseUnit は、基軸通貨（USD）1 単位あたりのセント数です。参考換算の入力金額へ換算します。
	centsPerBaseUnit = 100
)

// DetailParam は、購入明細の入力（商品 ID と数量）です。
type DetailParam struct {
	ProductID uuid.UUID
	Quantity  int
}

// CreatePurchaseParams は、購入作成の入力パラメータです。
type CreatePurchaseParams struct {
	// UserID は、購入者（認証済みの内部ユーザー ID）です。
	UserID uuid.UUID
	// Details は、購入明細の配列です。
	Details []DetailParam
	// DisplayCurrency は、参考換算額の表示通貨です。nil の場合は参考換算額を返しません。
	DisplayCurrency *string
}

// PurchaseDetailView は、購入明細のユースケース出力 DTO です。
type PurchaseDetailView struct {
	ProductID uuid.UUID
	Quantity  int
	UnitPrice int
}

// ReferenceAmountView は、参考換算額のユースケース出力 DTO です（非永続）。
type ReferenceAmountView struct {
	Currency string
	Amount   int64
	Rate     float64
	RateDate string
}

// PurchaseView は、購入 1 件分のユースケース出力 DTO です。金額はすべて USD セント単位の整数です。
type PurchaseView struct {
	ID              uuid.UUID
	Code            string
	UserID          uuid.UUID
	StatusID        uuid.UUID
	SubtotalAmount  int
	TaxAmount       int
	ShippingFee     int
	TotalAmount     int
	Details         []PurchaseDetailView
	OrderedAt       time.Time
	ReferenceAmount *ReferenceAmountView
}

// Usecase は、購入の作成ユースケースを定義します。
type Usecase interface {
	// CreatePurchase は、明細から購入を作成します。在庫減算・購入作成・明細作成・outbox 発行を単一 tx で
	// 原子的に行い、売り越しは 409 で成立させません。DisplayCurrency 指定時は参考換算額を付与します。
	CreatePurchase(ctx context.Context, params CreatePurchaseParams) (PurchaseView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	txm    tx.Manager
	cmd    command.CommandService
	repo   purchase.Repository
	emit   outbox.EmitUsecase
	xr     exchangerate.Usecase
}

// New は、購入の作成ユースケースを生成します。
func New(
	txm tx.Manager,
	cmd command.CommandService,
	repo purchase.Repository,
	emit outbox.EmitUsecase,
	xr exchangerate.Usecase,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		txm:    txm,
		cmd:    cmd,
		repo:   repo,
		emit:   emit,
		xr:     xr,
	}
}

// CreatePurchase は、明細から購入を作成します。
func (u *usecase) CreatePurchase(ctx context.Context, params CreatePurchaseParams) (PurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	purchaseID, err := uuid.New()
	if err != nil {
		return PurchaseView{}, xerrors.Wrap(err, "failed to generate purchase id")
	}
	codeUUID, err := uuid.New()
	if err != nil {
		return PurchaseView{}, xerrors.Wrap(err, "failed to generate purchase code")
	}
	code := codeUUID.String()

	inputs := make([]purchase.DetailInput, len(params.Details))
	productIDs := make([]uuid.UUID, len(params.Details))
	for i, d := range params.Details {
		detailID, derr := uuid.New()
		if derr != nil {
			return PurchaseView{}, xerrors.Wrap(derr, "failed to generate purchase detail id")
		}
		inputs[i] = purchase.DetailInput{ID: detailID, ProductID: d.ProductID, Quantity: d.Quantity}
		productIDs[i] = d.ProductID
	}

	var created *purchase.Purchase
	// 最外 tx は idempotency.Run が所有する。ここは nested で同一 tx に乗り、部分適用を防ぐ。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		locked, lerr := u.cmd.LockProducts(ctx, productIDs)
		if lerr != nil {
			return lerr
		}

		entity, nerr := purchase.New(purchaseID, code, params.UserID, inputs, locked)
		if nerr != nil {
			return nerr
		}

		if cerr := u.cmd.CreatePurchase(ctx, entity); cerr != nil {
			return cerr
		}

		payload, perr := buildCreatedPayload(entity)
		if perr != nil {
			return perr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   purchaseID.String(),
			EventType:     eventTypeCreated,
			Payload:       payload,
		}); eerr != nil {
			return eerr
		}

		// 書き込み後、Repository 経由でドメイン整合を再検証しレスポンスの取得元とする（ADR-0027 / ADR-0029）。
		reread, rerr := u.repo.FindByID(ctx, purchaseID)
		if rerr != nil {
			return rerr
		}
		created = reread
		return nil
	}); txErr != nil {
		return PurchaseView{}, txErr
	}

	view := toPurchaseView(created)
	// 参考換算額は tx 外（レスポンス組み立て時）に算出し、取得失敗時は null で degrade する。
	if params.DisplayCurrency != nil {
		view.ReferenceAmount = u.referenceAmount(ctx, created.TotalAmount(), *params.DisplayCurrency)
	}
	return view, nil
}

// referenceAmount は、合計金額（USD セント）の表示通貨での参考換算額を算出します。
// 為替 gateway の障害時は nil を返して degrade します（購入は既に成立している）。
func (u *usecase) referenceAmount(ctx context.Context, totalCents int, displayCurrency string) *ReferenceAmountView {
	res, err := u.xr.Convert(ctx, exchangerate.ConvertInput{
		Base:            baseCurrency,
		Quote:           displayCurrency,
		Amount:          float64(totalCents) / float64(centsPerBaseUnit),
		DisplayCurrency: &displayCurrency,
	})
	if err != nil || res.Reference == nil {
		return nil
	}
	return &ReferenceAmountView{
		Currency: res.Reference.Currency,
		Amount:   res.Reference.Amount,
		Rate:     res.Reference.Rate,
		RateDate: res.Reference.RateDate,
	}
}

// toPurchaseView は、購入集約を出力 DTO へ変換します。
func toPurchaseView(p *purchase.Purchase) PurchaseView {
	details := p.Details()
	views := make([]PurchaseDetailView, len(details))
	for i, d := range details {
		views[i] = PurchaseDetailView{
			ProductID: d.ProductID(),
			Quantity:  d.Quantity(),
			UnitPrice: d.UnitPrice(),
		}
	}
	return PurchaseView{
		ID:             p.ID(),
		Code:           p.Code(),
		UserID:         p.UserID(),
		StatusID:       p.StatusID(),
		SubtotalAmount: p.SubtotalAmount(),
		TaxAmount:      p.TaxAmount(),
		ShippingFee:    p.ShippingFee(),
		TotalAmount:    p.TotalAmount(),
		Details:        views,
		OrderedAt:      p.OrderedAt(),
	}
}

// buildCreatedPayload は、purchase.created.v1 の自己完結 snapshot payload を組み立てます（ADR-0043）。
func buildCreatedPayload(p *purchase.Purchase) ([]byte, error) {
	type detail struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
		UnitPrice int    `json:"unitPrice"`
	}
	type event struct {
		PurchaseID     string   `json:"purchaseId"`
		Code           string   `json:"code"`
		UserID         string   `json:"userId"`
		StatusCode     int      `json:"statusCode"`
		SubtotalAmount int      `json:"subtotalAmount"`
		TaxAmount      int      `json:"taxAmount"`
		ShippingFee    int      `json:"shippingFee"`
		TotalAmount    int      `json:"totalAmount"`
		Details        []detail `json:"details"`
	}

	src := p.Details()
	details := make([]detail, len(src))
	for i, d := range src {
		details[i] = detail{
			ProductID: d.ProductID().String(),
			Quantity:  d.Quantity(),
			UnitPrice: d.UnitPrice(),
		}
	}

	payload, err := json.Marshal(event{
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
