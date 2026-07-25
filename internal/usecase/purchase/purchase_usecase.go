//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package purchase は、購入の作成ユースケースを提供します。単価は価格スケール（ドル decimal）、
// 決済額は決済スケール（整数セント）で扱います（ADR-0101 / ADR-0102）。
package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/internal/usecase/exchangerate"
	"go-boilerplate/internal/usecase/outbox"
	"go-boilerplate/internal/usecase/purchase/command"
	"go-boilerplate/internal/usecase/purchase/event"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// baseCurrency は、購入金額の基軸通貨です。金額は本通貨のセント整数で保持します。
	baseCurrency = "USD"
	// aggregateType は、outbox の集約種別です。
	aggregateType = "purchase"
	// centsPerBaseUnit は、基軸通貨（USD）1 単位あたりのセント数です。参考換算の入力金額へ換算します。
	centsPerBaseUnit = 100
	// minorUnitDigits は、基軸通貨（USD）の最小単位の小数桁数です（セント = 小数 2 桁）。
	minorUnitDigits = 2
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

// PurchaseDetailView は、購入明細のユースケース出力 DTO です。UnitPrice は価格スケール（ドル decimal）です。
type PurchaseDetailView struct {
	ProductID uuid.UUID
	Quantity  int
	UnitPrice decimal.Decimal
}

// ReferenceAmountView は、参考換算額のユースケース出力 DTO です（非永続）。
type ReferenceAmountView struct {
	Currency string
	Amount   int64
	Rate     decimal.Decimal
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

// CancelPurchaseParams は、購入キャンセルの入力パラメータです。
type CancelPurchaseParams struct {
	// PurchaseID は、キャンセル対象の購入 ID です。
	PurchaseID uuid.UUID
	// UserID は、キャンセルを要求した認証済みの内部ユーザー ID です。所有権の検証に用います。
	UserID uuid.UUID
}

// CancelPurchaseView は、キャンセル後の購入 1 件分のユースケース出力 DTO です。ステータスは購入ステータス
// マスタで解決済みの ID と名称、CanceledAt はキャンセル日時です。金額はすべて USD セント単位の整数です。
type CancelPurchaseView struct {
	ID             uuid.UUID
	Code           string
	UserID         uuid.UUID
	StatusID       uuid.UUID
	StatusName     string
	SubtotalAmount int
	TaxAmount      int
	ShippingFee    int
	TotalAmount    int
	Details        []PurchaseDetailView
	OrderedAt      time.Time
	CanceledAt     *time.Time
}

// PayPurchaseParams は、購入支払いの入力パラメータです。
type PayPurchaseParams struct {
	// PurchaseID は、支払い対象の購入 ID です。
	PurchaseID uuid.UUID
	// UserID は、支払いを要求した認証済みの内部ユーザー ID です。所有権の検証に用います。
	UserID uuid.UUID
}

// PayPurchaseView は、支払い後の購入 1 件分のユースケース出力 DTO です。ステータスは購入ステータス
// マスタで解決済みの ID と名称、PaidAt は支払い日時です。金額はすべて USD セント単位の整数です。
type PayPurchaseView struct {
	ID             uuid.UUID
	Code           string
	UserID         uuid.UUID
	StatusID       uuid.UUID
	StatusName     string
	SubtotalAmount int
	TaxAmount      int
	ShippingFee    int
	TotalAmount    int
	Details        []PurchaseDetailView
	OrderedAt      time.Time
	PaidAt         *time.Time
}

// Usecase は、購入の作成ユースケースを定義します。
type Usecase interface {
	// CreatePurchase は、明細から購入を作成します。在庫減算・購入作成・明細作成・outbox 発行を単一 tx で
	// 原子的に行い、売り越しは 409 で成立させません。DisplayCurrency 指定時は参考換算額を付与します。
	CreatePurchase(ctx context.Context, params CreatePurchaseParams) (PurchaseView, error)
	// GetPurchases は、認証主体（userID）の購入履歴を注文日時降順（cursor ページネーション）で取得します。
	// 一覧は概要（code / totalAmount / status / orderedAt）のみを返し、他ユーザーの購入は返しません。
	GetPurchases(ctx context.Context, userID uuid.UUID, cursor *paging.Cursor) (*PurchaseListView, error)
	// CancelPurchase は、本人の購入をキャンセルします。購入行のロック → 所有権検証 → 状態遷移（→ キャンセル）→
	// 明細分の在庫復元 → outbox 発行を単一 tx で原子的に行います。他ユーザーの購入・不存在はいずれも
	// 存在秘匿のため NotFound（404）、不正遷移は 409 を返します。
	CancelPurchase(ctx context.Context, params CancelPurchaseParams) (CancelPurchaseView, error)
	// PayPurchase は、本人の購入を支払い済みへ遷移させます。購入行のロック → 所有権検証 → 状態遷移（→ 支払い済み）→
	// outbox 発行を単一 tx で原子的に行います。決済 SDK / PSP 連携は行わない擬似決済です。他ユーザーの購入・不存在は
	// いずれも存在秘匿のため NotFound（404）、二重支払い・不正遷移は 409 を返します。
	PayPurchase(ctx context.Context, params PayPurchaseParams) (PayPurchaseView, error)
	// GetPurchaseDetail は、本人の購入 1 件を明細（商品名込み）とともに取得します。QueryService の集約跨ぎ read 投影を用い、
	// 所有権は QS の SQL 述語で担保します。他ユーザーの購入・不存在はいずれも存在秘匿のため NotFound（404）を返します。
	GetPurchaseDetail(ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID) (PurchaseGetDetailView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer   observability.LayerTracer
	txm      tx.Manager
	cmd      command.CommandService
	repo     purchase.Repository
	detailQS query.PurchaseDetailQueryService
	emit     outbox.EmitUsecase
	xr       exchangerate.Usecase
	clock    clock.Clock
}

// New は、購入ユースケースを生成します。
func New(
	txm tx.Manager,
	cmd command.CommandService,
	repo purchase.Repository,
	detailQS query.PurchaseDetailQueryService,
	emit outbox.EmitUsecase,
	xr exchangerate.Usecase,
	clock clock.Clock,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer:   tf.Usecase(),
		txm:      txm,
		cmd:      cmd,
		repo:     repo,
		detailQS: detailQS,
		emit:     emit,
		xr:       xr,
		clock:    clock,
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

		payload, perr := event.BuildCreated(entity)
		if perr != nil {
			return perr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   purchaseID.String(),
			EventType:     event.TypeCreated,
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

// CancelPurchase は、本人の購入をキャンセルします。
func (u *usecase) CancelPurchase(ctx context.Context, params CancelPurchaseParams) (CancelPurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	var detail *purchase.Detail
	// この Do が最外 tx（本エンドポイントは Idempotency-Key 冪等化を配線しない）。状態遷移と在庫復元を
	// 単一 tx にまとめ部分適用を防ぐ。二重キャンセルは購入行ロック + 状態チェック（ErrAlreadyCanceled）で安全化する。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		locked, lerr := u.cmd.LockPurchase(ctx, params.PurchaseID)
		if lerr != nil {
			return lerr
		}

		// 他人の購入は存在を秘匿するため、不一致・不存在いずれも NotFound（404）へ畳む。
		if locked.UserID() != params.UserID {
			return xerrors.Wrap(apperror.ErrNotFound, "purchase not found")
		}

		if cerr := locked.Cancel(now); cerr != nil {
			return cerr
		}

		if perr := u.cmd.CancelPurchase(ctx, locked); perr != nil {
			return perr
		}

		payload, berr := event.BuildCanceled(locked)
		if berr != nil {
			return berr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   params.PurchaseID.String(),
			EventType:     event.TypeCanceled,
			Payload:       payload,
		}); eerr != nil {
			return eerr
		}

		// 書き込み後、Repository の読み取りモデル経由でステータス名を解決しレスポンスの取得元とする（ADR-0027 / ADR-0029）。
		reread, rerr := u.repo.FindDetailByID(ctx, params.PurchaseID)
		if rerr != nil {
			return rerr
		}
		detail = reread
		return nil
	}); txErr != nil {
		return CancelPurchaseView{}, txErr
	}

	return toCancelPurchaseView(detail), nil
}

// PayPurchase は、本人の購入を支払い済みへ遷移させます。
func (u *usecase) PayPurchase(ctx context.Context, params PayPurchaseParams) (PayPurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	var detail *purchase.Detail
	// この Do が最外 tx（本エンドポイントは Idempotency-Key 冪等化を配線しない）。擬似決済は単一集約
	// （purchases）の状態更新のみで在庫に触れないため CommandService ではなく Repository で完結する。
	// 二重支払いは購入行ロック + 状態チェック（ErrAlreadyPaid）で安全化する。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		locked, lerr := u.repo.LockByID(ctx, params.PurchaseID)
		if lerr != nil {
			return lerr
		}

		// 他人の購入は存在を秘匿するため、不一致・不存在いずれも NotFound（404）へ畳む。
		if locked.UserID() != params.UserID {
			return xerrors.Wrap(apperror.ErrNotFound, "purchase not found")
		}

		if perr := locked.Pay(now); perr != nil {
			return perr
		}

		if perr := u.repo.UpdatePaid(ctx, locked); perr != nil {
			return perr
		}

		payload, berr := event.BuildPaid(locked)
		if berr != nil {
			return berr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   params.PurchaseID.String(),
			EventType:     event.TypePaid,
			Payload:       payload,
		}); eerr != nil {
			return eerr
		}

		// 書き込み後、Repository の読み取りモデル経由でステータス名を解決しレスポンスの取得元とする（ADR-0027 / ADR-0029）。
		reread, rerr := u.repo.FindDetailByID(ctx, params.PurchaseID)
		if rerr != nil {
			return rerr
		}
		detail = reread
		return nil
	}); txErr != nil {
		return PayPurchaseView{}, txErr
	}

	return toPayPurchaseView(detail), nil
}

// referenceAmount は、合計金額（USD セント）の表示通貨での参考換算額を算出します。
// 為替 gateway の障害時は nil を返して degrade します（購入は既に成立している）。
func (u *usecase) referenceAmount(ctx context.Context, totalCents int, displayCurrency string) *ReferenceAmountView {
	// 決済スケール（整数セント）の合計を価格スケール（ドル decimal）へ戻して換算入力にする。
	amount := decimal.FromInt(int64(totalCents)).DivRound(decimal.FromInt(centsPerBaseUnit), minorUnitDigits)
	res, err := u.xr.Convert(ctx, exchangerate.ConvertInput{
		Base:            baseCurrency,
		Quote:           displayCurrency,
		Amount:          amount,
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
			UnitPrice: d.UnitPrice().Decimal(),
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

// toCancelPurchaseView は、購入詳細の読み取りモデルをキャンセルレスポンスの出力 DTO へ変換します。
func toCancelPurchaseView(d *purchase.Detail) CancelPurchaseView {
	views := make([]PurchaseDetailView, len(d.Details))
	for i, detail := range d.Details {
		views[i] = PurchaseDetailView{
			ProductID: detail.ProductID(),
			Quantity:  detail.Quantity(),
			UnitPrice: detail.UnitPrice().Decimal(),
		}
	}
	return CancelPurchaseView{
		ID:             d.ID,
		Code:           d.Code,
		UserID:         d.UserID,
		StatusID:       d.StatusID,
		StatusName:     d.StatusName,
		SubtotalAmount: d.SubtotalAmount,
		TaxAmount:      d.TaxAmount,
		ShippingFee:    d.ShippingFee,
		TotalAmount:    d.TotalAmount,
		Details:        views,
		OrderedAt:      d.OrderedAt,
		CanceledAt:     d.CanceledAt,
	}
}

// toPayPurchaseView は、購入詳細の読み取りモデルを支払いレスポンスの出力 DTO へ変換します。
func toPayPurchaseView(d *purchase.Detail) PayPurchaseView {
	views := make([]PurchaseDetailView, len(d.Details))
	for i, detail := range d.Details {
		views[i] = PurchaseDetailView{
			ProductID: detail.ProductID(),
			Quantity:  detail.Quantity(),
			UnitPrice: detail.UnitPrice().Decimal(),
		}
	}
	return PayPurchaseView{
		ID:             d.ID,
		Code:           d.Code,
		UserID:         d.UserID,
		StatusID:       d.StatusID,
		StatusName:     d.StatusName,
		SubtotalAmount: d.SubtotalAmount,
		TaxAmount:      d.TaxAmount,
		ShippingFee:    d.ShippingFee,
		TotalAmount:    d.TotalAmount,
		Details:        views,
		OrderedAt:      d.OrderedAt,
		PaidAt:         d.PaidAt,
	}
}
