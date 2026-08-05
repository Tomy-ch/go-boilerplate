//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package purchase は、購入の作成ユースケースを提供します。単価は価格スケール（ドル decimal）、
// 決済額は決済スケール（整数セント）で扱います（ADR-0101 / ADR-0102）。
package purchase

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/domain/purchase"
	"go-boilerplate/internal/domain/service/membership"
	"go-boilerplate/internal/domain/user"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/tx"
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
	// aggregateType は、outbox の集約種別です。
	aggregateType = "purchase"
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
}

// PurchaseDetailView は、購入明細のユースケース出力 DTO です。UnitPrice は価格スケール（ドル decimal）です。
type PurchaseDetailView struct {
	ProductID uuid.UUID
	Quantity  int
	UnitPrice decimal.Decimal
}

// PurchaseView は、購入 1 件分のユースケース出力 DTO です。金額はすべて USD セント単位の整数です。
type PurchaseView struct {
	ID             uuid.UUID
	Code           string
	UserID         uuid.UUID
	StatusID       uuid.UUID
	SubtotalAmount int
	TaxAmount      int
	ShippingFee    int
	TotalAmount    int
	Details        []PurchaseDetailView
	OrderedAt      time.Time
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

// ShipPurchaseView は、発送後の購入 1 件分のユースケース出力 DTO です。ステータスは購入ステータス
// マスタで解決済みの ID と名称、ShippedAt は発送日時です。金額はすべて USD セント単位の整数です。
type ShipPurchaseView struct {
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
	ShippedAt      *time.Time
}

// DeliverPurchaseView は、配達完了後の購入 1 件分のユースケース出力 DTO です。ステータスは購入ステータス
// マスタで解決済みの ID と名称、DeliveredAt は配達日時です。金額はすべて USD セント単位の整数です。
type DeliverPurchaseView struct {
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
	DeliveredAt    *time.Time
}

// Usecase は、購入の作成ユースケースを定義します。
type Usecase interface {
	// CreatePurchase は、明細から購入を作成します。在庫の引当・購入の成立・イベント発行は単一 tx で
	// 原子的に成立し、売り越しは 409 で成立させません。DisplayCurrency 指定時は参考換算額を付与します。
	CreatePurchase(ctx context.Context, params CreatePurchaseParams) (PurchaseView, error)
	// GetPurchases は、認証主体（userID）の購入履歴を注文日時降順（cursor ページネーション）で取得します。
	// 一覧は概要（code / totalAmount / status / orderedAt）のみを返し、他ユーザーの購入は返しません。
	GetPurchases(ctx context.Context, userID uuid.UUID, cursor *paging.Cursor) (*PurchaseListView, error)
	// CancelPurchase は、本人の購入をキャンセルし、明細分の在庫を復元します。キャンセル・在庫復元・
	// イベント発行は単一 tx で原子的に成立します。他ユーザーの購入・不存在はいずれも存在秘匿のため
	// NotFound（404）、不正遷移は 409 を返します。
	CancelPurchase(ctx context.Context, params CancelPurchaseParams) (CancelPurchaseView, error)
	// PayPurchase は、本人の購入を支払い済みへ遷移させます。決済 SDK / PSP 連携は行わない擬似決済です。
	// 状態遷移とイベント発行は単一 tx で原子的に成立します。他ユーザーの購入・不存在はいずれも存在秘匿のため
	// NotFound（404）、二重支払い・不正遷移は 409 を返します。
	PayPurchase(ctx context.Context, params PayPurchaseParams) (PayPurchaseView, error)
	// ShipPurchase は、購入を発送済みへ遷移させます。実行できるのは管理者のみで、購入の所有者であるかは問いません。
	// 状態遷移とイベント発行は単一 tx で原子的に成立します。管理者でない場合は 403、不存在は 404、
	// 二重発送・不正遷移は 409 を返します。
	ShipPurchase(ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID) (ShipPurchaseView, error)
	// DeliverPurchase は、購入を配達済みへ遷移させます。実行できるのは管理者のみで、購入の所有者であるかは問いません。
	// 状態遷移とイベント発行は単一 tx で原子的に成立します。管理者でない場合は 403、不存在は 404、
	// 二重配達・不正遷移は 409 を返します。
	DeliverPurchase(ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID) (DeliverPurchaseView, error)
	// GetPurchaseDetail は、本人の購入 1 件を明細（商品名込み）とともに取得します。
	// 他ユーザーの購入・不存在はいずれも存在秘匿のため NotFound（404）を返します。
	GetPurchaseDetail(ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID) (PurchaseGetDetailView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer      observability.LayerTracer
	txm         tx.Manager
	cmd         command.CommandService
	repo        purchase.Repository
	productRepo product.Repository
	userLock    user.LockRepository
	detailQS    query.PurchaseDetailQueryService
	emit        outbox.EmitUsecase
	clock       clock.Clock
	authorizer  authz.Authorizer
}

// purchaseDraft は、購入作成のトランザクションへ持ち込む採番済みの入力です。
type purchaseDraft struct {
	purchaseID uuid.UUID
	code       string
	inputs     []purchase.DetailInput
	productIDs []uuid.UUID
}

// New は、購入ユースケースを生成します。
func New(
	txm tx.Manager,
	cmd command.CommandService,
	repo purchase.Repository,
	productRepo product.Repository,
	userLock user.LockRepository,
	detailQS query.PurchaseDetailQueryService,
	emit outbox.EmitUsecase,
	clock clock.Clock,
	authorizer authz.Authorizer,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer:      tf.Usecase(),
		txm:         txm,
		cmd:         cmd,
		repo:        repo,
		productRepo: productRepo,
		userLock:    userLock,
		detailQS:    detailQS,
		emit:        emit,
		clock:       clock,
		authorizer:  authorizer,
	}
}

// newPurchaseDraft は、購入・購入コード・各明細の ID を採番し、ドメイン入力と商品 ID 列を組み立てます。
// 採番はトランザクションの外で行うため、リトライされても同じ ID が再利用されることはありません。
func newPurchaseDraft(details []DetailParam) (*purchaseDraft, error) {
	purchaseID, err := uuid.New()
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to generate purchase id")
	}
	codeUUID, err := uuid.New()
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to generate purchase code")
	}

	draft := &purchaseDraft{
		purchaseID: purchaseID,
		code:       codeUUID.String(),
		inputs:     make([]purchase.DetailInput, len(details)),
		productIDs: make([]uuid.UUID, len(details)),
	}
	for i, d := range details {
		detailID, derr := uuid.New()
		if derr != nil {
			return nil, xerrors.Wrap(derr, "failed to generate purchase detail id")
		}
		draft.inputs[i] = purchase.DetailInput{ID: detailID, ProductID: d.ProductID, Quantity: d.Quantity}
		draft.productIDs[i] = d.ProductID
	}
	return draft, nil
}

func (u *usecase) CreatePurchase(ctx context.Context, params CreatePurchaseParams) (PurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	draft, err := newPurchaseDraft(params.Details)
	if err != nil {
		return PurchaseView{}, err
	}

	var created *purchase.Purchase
	// 最外 tx は idempotency.Run が所有する。ここは nested で同一 tx に乗り、部分適用を防ぐ。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		// ロックはユーザー行 → 商品行（id 昇順）の順で取り、順序を全 tx で固定してデッドロックを避ける。
		if uerr := u.ensurePurchaserActive(ctx, params.UserID); uerr != nil {
			return uerr
		}

		// 在庫ロックは商品集約の読み取りなので、購入の書き込みポートではなく商品 Repository を通す。
		products, lerr := u.productRepo.LockByIDs(ctx, draft.productIDs)
		if lerr != nil {
			return lerr
		}
		locked := make([]purchase.LockedProduct, len(products))
		for i, p := range products {
			locked[i] = purchase.NewLockedProduct(p.ID(), p.Price(), p.Quantity())
		}

		entity, nerr := purchase.New(draft.purchaseID, draft.code, params.UserID, draft.inputs, locked)
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
			AggregateID:   draft.purchaseID.String(),
			EventType:     event.TypeCreated,
			Payload:       payload,
		}); eerr != nil {
			return eerr
		}

		// 書き込み後、Repository 経由でドメイン整合を再検証しレスポンスの取得元とする（ADR-0027 / ADR-0029）。
		reread, rerr := u.repo.FindByID(ctx, draft.purchaseID)
		if rerr != nil {
			return rerr
		}
		created = reread
		return nil
	}); txErr != nil {
		return PurchaseView{}, txErr
	}

	return toPurchaseView(created), nil
}

func (u *usecase) CancelPurchase(ctx context.Context, params CancelPurchaseParams) (CancelPurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	var detail *purchase.Detail
	// この Do が最外 tx（本エンドポイントは Idempotency-Key 冪等化を配線しない）。状態遷移と在庫復元を
	// 単一 tx にまとめ部分適用を防ぐ。二重キャンセルは購入のロック + 状態チェック（ErrAlreadyCanceled）で安全化する。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		locked, lerr := u.cmd.LockPurchase(ctx, params.PurchaseID)
		if lerr != nil {
			return lerr
		}

		// 他人の購入は存在を秘匿するため、不一致・不存在いずれも NotFound（404）へ畳む。
		if locked.UserID() != params.UserID {
			return xerrors.Wrap(apperror.ErrNotFound, "purchase not found")
		}

		domainEvent, cerr := locked.Cancel(now)
		if cerr != nil {
			return cerr
		}

		if perr := u.cmd.CancelPurchase(ctx, locked); perr != nil {
			return perr
		}

		payload, berr := event.BuildCanceled(locked)
		if berr != nil {
			return berr
		}
		eventType, terr := event.WireType(domainEvent.Type())
		if terr != nil {
			return terr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   params.PurchaseID.String(),
			EventType:     eventType,
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

func (u *usecase) PayPurchase(ctx context.Context, params PayPurchaseParams) (PayPurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	now := u.clock.Now()

	var detail *purchase.Detail
	// この Do が最外 tx（本エンドポイントは Idempotency-Key 冪等化を配線しない）。擬似決済は購入集約の
	// 状態更新のみで在庫に触れないため CommandService ではなく Repository で完結する。
	// 二重支払いは購入のロック + 状態チェック（ErrAlreadyPaid）で安全化する。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		locked, lerr := u.repo.LockByID(ctx, params.PurchaseID)
		if lerr != nil {
			return lerr
		}

		// 他人の購入は存在を秘匿するため、不一致・不存在いずれも NotFound（404）へ畳む。
		if locked.UserID() != params.UserID {
			return xerrors.Wrap(apperror.ErrNotFound, "purchase not found")
		}

		domainEvent, perr := locked.Pay(now)
		if perr != nil {
			return perr
		}

		if perr := u.repo.UpdatePaid(ctx, locked); perr != nil {
			return perr
		}

		payload, berr := event.BuildPaid(locked)
		if berr != nil {
			return berr
		}
		eventType, terr := event.WireType(domainEvent.Type())
		if terr != nil {
			return terr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   params.PurchaseID.String(),
			EventType:     eventType,
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

func (u *usecase) ShipPurchase(
	ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID,
) (ShipPurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return ShipPurchaseView{}, apperror.ErrUnauthenticated
	}
	// 発送は運用（admin）の操作で所有者概念を持たないため、所有者なしリソースとして認可する。
	// 所有者を渡さないことで Authorizer の所有者フォールバックが働かず、admin だけが通る。
	if err := u.authorizer.Authorize(
		ctx, authn, authz.ActionPurchaseShip, authz.NewResource("purchase", nil),
	); err != nil {
		return ShipPurchaseView{}, err
	}

	now := u.clock.Now()

	var detail *purchase.Detail
	// この Do が最外 tx（本エンドポイントは Idempotency-Key 冪等化を配線しない）。配送追跡を扱わず単一集約
	// （purchases）の状態更新のみのため CommandService ではなく Repository で完結する。
	// 二重発送は購入行ロック + 状態チェック（ErrAlreadyShipped）で安全化する。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		locked, lerr := u.repo.LockByID(ctx, purchaseID)
		if lerr != nil {
			return lerr
		}

		domainEvent, serr := locked.Ship(now)
		if serr != nil {
			return serr
		}

		if uerr := u.repo.UpdateShipped(ctx, locked); uerr != nil {
			return uerr
		}

		payload, berr := event.BuildShipped(locked)
		if berr != nil {
			return berr
		}
		eventType, terr := event.WireType(domainEvent.Type())
		if terr != nil {
			return terr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   purchaseID.String(),
			EventType:     eventType,
			Payload:       payload,
		}); eerr != nil {
			return eerr
		}

		// 書き込み後、Repository の読み取りモデル経由でステータス名を解決しレスポンスの取得元とする（ADR-0027 / ADR-0029）。
		reread, rerr := u.repo.FindDetailByID(ctx, purchaseID)
		if rerr != nil {
			return rerr
		}
		detail = reread
		return nil
	}); txErr != nil {
		return ShipPurchaseView{}, txErr
	}

	return toShipPurchaseView(detail), nil
}

func (u *usecase) DeliverPurchase(
	ctx context.Context, authn *auth.Authn, purchaseID uuid.UUID,
) (DeliverPurchaseView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return DeliverPurchaseView{}, apperror.ErrUnauthenticated
	}
	// 配達完了は運用（admin）の操作で所有者概念を持たないため、所有者なしリソースとして認可する。
	// 所有者を渡さないことで Authorizer の所有者フォールバックが働かず、admin だけが通る。
	if err := u.authorizer.Authorize(
		ctx, authn, authz.ActionPurchaseDeliver, authz.NewResource("purchase", nil),
	); err != nil {
		return DeliverPurchaseView{}, err
	}

	now := u.clock.Now()

	var detail *purchase.Detail
	// この Do が最外 tx（本エンドポイントは Idempotency-Key 冪等化を配線しない）。配達確認の証跡を扱わず
	// 単一集約（purchases）の状態更新のみのため CommandService ではなく Repository で完結する。
	// 二重配達は購入行ロック + 状態チェック（ErrAlreadyDelivered）で安全化する。
	if txErr := u.txm.Do(ctx, func(ctx context.Context) error {
		locked, lerr := u.repo.LockByID(ctx, purchaseID)
		if lerr != nil {
			return lerr
		}

		domainEvent, derr := locked.Deliver(now)
		if derr != nil {
			return derr
		}

		if uerr := u.repo.UpdateDelivered(ctx, locked); uerr != nil {
			return uerr
		}

		payload, berr := event.BuildDelivered(locked)
		if berr != nil {
			return berr
		}
		eventType, terr := event.WireType(domainEvent.Type())
		if terr != nil {
			return terr
		}
		if _, eerr := u.emit.Emit(ctx, outbox.EmitInput{
			AggregateType: aggregateType,
			AggregateID:   purchaseID.String(),
			EventType:     eventType,
			Payload:       payload,
		}); eerr != nil {
			return eerr
		}

		// 書き込み後、Repository の読み取りモデル経由でステータス名を解決しレスポンスの取得元とする（ADR-0027 / ADR-0029）。
		reread, rerr := u.repo.FindDetailByID(ctx, purchaseID)
		if rerr != nil {
			return rerr
		}
		detail = reread
		return nil
	}); txErr != nil {
		return DeliverPurchaseView{}, txErr
	}

	return toDeliverPurchaseView(detail), nil
}

// ensurePurchaserActive は、購入者を共有ロック付きで読み出し、購入してよい状態かの判定を
// ドメインサービスへ委ねます。退会（排他ロック）と直列化されるため、確認を通った購入者は
// tx の終了まで退会できません。
// 退会が確定済みの主体は認証の時点で弾かれるため、ここが拒否に転じるのは受付から成立までの間に
// 退会が確定した場合に限られます。その衝突は主体の状態と操作の衝突として ErrConflict を返します
// （退会が進行中購入を ErrConflict で拒む鏡像。購入対象の不存在ではないため NotFound へは畳みません）。
// それ以外のエラーは、障害を退会済みと区別できなくしないためそのまま返します。
func (u *usecase) ensurePurchaserActive(ctx context.Context, userID uuid.UUID) error {
	purchaser, err := u.userLock.LockShareByID(ctx, userID)
	if err != nil {
		if xerrors.Is(err, apperror.ErrNotFound) {
			return membership.ErrPurchaserWithdrawn
		}
		return err
	}
	return membership.EnsurePurchasable(purchaser)
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

// toShipPurchaseView は、購入詳細の読み取りモデルを発送レスポンスの出力 DTO へ変換します。
func toShipPurchaseView(d *purchase.Detail) ShipPurchaseView {
	views := make([]PurchaseDetailView, len(d.Details))
	for i, detail := range d.Details {
		views[i] = PurchaseDetailView{
			ProductID: detail.ProductID(),
			Quantity:  detail.Quantity(),
			UnitPrice: detail.UnitPrice().Decimal(),
		}
	}
	return ShipPurchaseView{
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
		ShippedAt:      d.ShippedAt,
	}
}

// toDeliverPurchaseView は、購入詳細の読み取りモデルを配達完了レスポンスの出力 DTO へ変換します。
func toDeliverPurchaseView(d *purchase.Detail) DeliverPurchaseView {
	views := make([]PurchaseDetailView, len(d.Details))
	for i, detail := range d.Details {
		views[i] = PurchaseDetailView{
			ProductID: detail.ProductID(),
			Quantity:  detail.Quantity(),
			UnitPrice: detail.UnitPrice().Decimal(),
		}
	}
	return DeliverPurchaseView{
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
		DeliveredAt:    d.DeliveredAt,
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
