// Package purchase は、購入ドメインを定義します。購入（Purchase）集約と明細（PurchaseDetail）を持ち、
// 金額計算・売り越し検証・単価スナップショットの不変条件を保持します。
//
// 金額は 2 スケールモデル（ADR-0101 / ADR-0102）で扱います。単価（unit_price）はドル主単位のサブセント可 decimal
// （価格スケール）を money.Price で保持し、決済額（小計・税・送料・合計）は整数セント（決済スケール）で保持します。
// 価格スケールから決済スケールへの丸めは切り捨てで、New 内の 1 箇所（小計算出）に集約します。
package purchase

import (
	"fmt"
	"time"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// DetailInput は、購入明細の入力です。ID は採番済みの明細 ID、ProductID は購入対象商品、Quantity は数量です。
type DetailInput struct {
	// ID は、採番済みの明細 ID です（UUIDv7）。
	ID uuid.UUID
	// ProductID は、購入対象の商品 ID です。
	ProductID uuid.UUID
	// Quantity は、購入数量です。
	Quantity int
}

// LockedProduct は、在庫ロック取得直後の商品スナップショットです。Price は単価スナップショットの元値（価格スケール）です。
type LockedProduct struct {
	id       uuid.UUID
	price    money.Price
	quantity int
}

// PurchaseDetail は、購入明細を表す値オブジェクトです。UnitPrice は購入時点の単価スナップショット（価格スケール）です。
type PurchaseDetail struct {
	id        uuid.UUID
	productID uuid.UUID
	quantity  int
	unitPrice money.Price
}

// Purchase は、購入を表すドメイン集約です。決済額（小計・税・送料・合計）は整数セント（決済スケール）で保持します。
// 状態機械の現在状態は statusCode（安定した業務キー）を source of truth とし、timestamps
// （canceledAt / shippedAt / deliveredAt）は「イベントがいつ起きたか」の監査記録として併用します。
type Purchase struct {
	id             uuid.UUID
	code           string
	userID         uuid.UUID
	statusID       uuid.UUID
	status         Status
	subtotalAmount int
	taxAmount      int
	shippingFee    int
	totalAmount    int
	details        []PurchaseDetail
	orderedAt      time.Time
	paidAt         *time.Time
	canceledAt     *time.Time
	shippedAt      *time.Time
	deliveredAt    *time.Time
}

// NewLockedProduct は、ロック済み商品スナップショットを生成します。price は価格スケール（ドル decimal）です。
func NewLockedProduct(id uuid.UUID, price money.Price, quantity int) LockedProduct {
	return LockedProduct{id: id, price: price, quantity: quantity}
}

// PurchaseDetailAttributes は、明細の再構築に必要な属性一式です。id と productID は隣接する
// 同じ uuid.UUID で、位置引数のままだと取り違えてもコンパイルも検証も通ってしまうため構造体で受けます。
type PurchaseDetailAttributes struct {
	ProductID uuid.UUID
	Quantity  int
	// UnitPrice は価格スケール（ドル decimal）です。
	UnitPrice money.Price
}

// NewPurchaseDetail は、永続化済みの明細を再構築します（Repository の読み出しで使用）。
func NewPurchaseDetail(id uuid.UUID, attrs PurchaseDetailAttributes) PurchaseDetail {
	return PurchaseDetail{
		id:        id,
		productID: attrs.ProductID,
		quantity:  attrs.Quantity,
		unitPrice: attrs.UnitPrice,
	}
}

// New は、購入明細の入力とロック済み在庫から購入集約を生成します。
// 明細が空、同一 productID が重複、数量が最小値未満、明細に対応するロック済み商品が無い場合はそれぞれ
// 検証エラー（422）を返し、要求数量がロック済み在庫を超える場合は ErrInsufficientStock（409）を返します。
// 単価は対応するロック済み商品の価格スナップショットとし、金額（小計・税・送料・合計）を整数で計算します
// （税・送料の丸めは切り捨てで本メソッド内 1 箇所に集約）。ステータスは「未処理」で生成します。
func New(
	id uuid.UUID,
	code string,
	userID uuid.UUID,
	inputs []DetailInput,
	locked []LockedProduct,
) (*Purchase, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if code == "" {
		return nil, xerrors.Wrap(ErrInvalidCode, "code is required")
	}
	if userID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidUserID, "userID is required")
	}
	if len(inputs) == 0 {
		return nil, ErrEmptyDetails
	}

	lockedByID := make(map[uuid.UUID]LockedProduct, len(locked))
	for _, l := range locked {
		lockedByID[l.id] = l
	}

	seen := make(map[uuid.UUID]struct{}, len(inputs))
	details := make([]PurchaseDetail, 0, len(inputs))
	// 小計は価格スケール（ドル decimal）で正確に積算し、最後に決済スケール（整数セント）へ切り捨てる。
	subtotalDollars := decimal.FromInt(0)
	for _, in := range inputs {
		if in.ID.IsNil() {
			return nil, xerrors.Wrap(ErrInvalidID, "detail id is required")
		}
		if in.Quantity < minQuantity {
			return nil, xerrors.Wrap(ErrInvalidQuantity, fmt.Sprintf("quantity must be %d or greater, got %d", minQuantity, in.Quantity))
		}
		if _, dup := seen[in.ProductID]; dup {
			return nil, xerrors.Wrap(ErrDuplicateProductID, fmt.Sprintf("product %s appears more than once", in.ProductID))
		}
		seen[in.ProductID] = struct{}{}

		l, ok := lockedByID[in.ProductID]
		if !ok {
			return nil, xerrors.Wrap(ErrProductNotFound, fmt.Sprintf("product %s is not available", in.ProductID))
		}
		if in.Quantity > l.quantity {
			return nil, xerrors.Wrap(
				ErrInsufficientStock,
				fmt.Sprintf("product %s: requested %d, in stock %d", in.ProductID, in.Quantity, l.quantity),
			)
		}

		details = append(details, PurchaseDetail{
			id:        in.ID,
			productID: in.ProductID,
			quantity:  in.Quantity,
			unitPrice: l.price,
		})
		subtotalDollars = subtotalDollars.Add(l.price.Decimal().Mul(decimal.FromInt(int64(in.Quantity))))
	}

	// 決済スケールへの丸め（切り捨て）はこの 1 箇所のみ。以降の税・合計は整数セントで計算する。
	subtotalCents, err := subtotalDollars.Truncate(minorUnitDigits).ToScaledInt64(minorUnitDigits)
	if err != nil {
		return nil, xerrors.Wrap(ErrInvalidAmount, "subtotal exceeds the settlement range: "+err.Error())
	}
	subtotal := int(subtotalCents)
	tax := subtotal * taxRatePercent / percentDivisor
	shipping := shippingFeeCents
	total := subtotal + tax + shipping

	return &Purchase{
		id:             id,
		code:           code,
		userID:         userID,
		status:         StatusUnprocessed,
		subtotalAmount: subtotal,
		taxAmount:      tax,
		shippingFee:    shipping,
		totalAmount:    total,
		details:        details,
	}, nil
}

// Reconstruct は、永続化済みの購入を再構築します（Repository の読み出し・書き込み後の再検証で使用）。
// statusCode は購入ステータスマスタで解決した現在状態、paidAt / canceledAt / shippedAt / deliveredAt は
// 各イベントの発生日時（未発生は nil）です。ID / code / userID / statusID が nil、statusCode が不正、
// 金額が負、明細が空の場合は検証エラーを返します。statusCode と timestamps の組み合わせが状態遷移では
// 到達し得ないもの（発送後のキャンセル、発送を伴わない配達 等）の場合も ErrInvalidStatusID を返します。
// Attributes は、購入の再構築に必要な属性一式です。金額 4 つは同じ int、イベント日時 4 つは
// 同じ *time.Time で、位置引数のままだと取り違えても検証の大半を通過してしまうため構造体で受けます。
// DB 行からの再構築はこの層で最も晒された呼び出し元です。
type Attributes struct {
	Code           string
	UserID         uuid.UUID
	StatusID       uuid.UUID
	StatusCode     int
	SubtotalAmount int
	TaxAmount      int
	ShippingFee    int
	TotalAmount    int
	Details        []PurchaseDetail
	OrderedAt      time.Time
	PaidAt         *time.Time
	CanceledAt     *time.Time
	ShippedAt      *time.Time
	DeliveredAt    *time.Time
}

func Reconstruct(id uuid.UUID, attrs Attributes) (*Purchase, error) {
	if id.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidID, "id is required")
	}
	if attrs.Code == "" {
		return nil, xerrors.Wrap(ErrInvalidCode, "code is required")
	}
	if attrs.UserID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidUserID, "userID is required")
	}
	if attrs.StatusID.IsNil() {
		return nil, xerrors.Wrap(ErrInvalidStatusID, "statusID is required")
	}
	status, err := NewStatus(attrs.StatusCode)
	if err != nil {
		return nil, err
	}
	if err := validateStatusTimestamps(
		status, attrs.PaidAt, attrs.CanceledAt, attrs.ShippedAt, attrs.DeliveredAt,
	); err != nil {
		return nil, err
	}
	if attrs.SubtotalAmount < 0 || attrs.TaxAmount < 0 || attrs.ShippingFee < 0 || attrs.TotalAmount < 0 {
		return nil, xerrors.Wrap(ErrInvalidAmount, "amounts must not be negative")
	}
	if len(attrs.Details) == 0 {
		return nil, ErrEmptyDetails
	}

	copied := make([]PurchaseDetail, len(attrs.Details))
	copy(copied, attrs.Details)

	return &Purchase{
		id:             id,
		code:           attrs.Code,
		userID:         attrs.UserID,
		statusID:       attrs.StatusID,
		status:         status,
		subtotalAmount: attrs.SubtotalAmount,
		taxAmount:      attrs.TaxAmount,
		shippingFee:    attrs.ShippingFee,
		totalAmount:    attrs.TotalAmount,
		details:        copied,
		orderedAt:      attrs.OrderedAt,
		paidAt:         ptr.Copy(attrs.PaidAt),
		canceledAt:     ptr.Copy(attrs.CanceledAt),
		shippedAt:      ptr.Copy(attrs.ShippedAt),
		deliveredAt:    ptr.Copy(attrs.DeliveredAt),
	}, nil
}

// validateStatusTimestamps は、statusCode と各イベント日時の組み合わせが状態遷移で到達し得るかを検証します。
// 到達し得ない組み合わせ（発送後のキャンセル、発送を伴わない配達 等）の場合は ErrInvalidStatusID を返します。
func validateStatusTimestamps(status Status, paidAt, canceledAt, shippedAt, deliveredAt *time.Time) error {
	// キャンセル status と canceledAt は同時セットの不変条件を持つ。片方のみの矛盾した永続化状態を弾く。
	if (status == StatusCanceled) != (canceledAt != nil) {
		return xerrors.Wrap(ErrInvalidStatusID, "canceled status and canceledAt must be consistent")
	}
	// 支払い済み status は paidAt を必須とする（一方向）。paidAt は支払い後に残り続け、以降のキャンセル
	// 等の遷移で status が変わっても保持されるため、キャンセルのような双条件にはしない。
	if status == StatusPaid && paidAt == nil {
		return xerrors.Wrap(ErrInvalidStatusID, "paid status requires paidAt")
	}
	// 発送済み status は shippedAt を必須とする（一方向）。shippedAt は発送後に残り続け、以降の
	// 配達済み等の遷移で status が変わっても保持されるため、キャンセルのような双条件にはしない。
	if status == StatusShipped && shippedAt == nil {
		return xerrors.Wrap(ErrInvalidStatusID, "shipped status requires shippedAt")
	}
	// キャンセルは発送前にしか行えない（Cancel が発送済み・配達済みを拒否する）ため、キャンセル status と
	// 発送の記録は同居しない。支払い後のキャンセルは正常なので paidAt は対象外。
	if status == StatusCanceled && shippedAt != nil {
		return xerrors.Wrap(ErrInvalidStatusID, "canceled status must not have shippedAt")
	}
	// 発送済みへは支払い済みからのみ遷移し、支払い済みは paidAt を必須とするため、paidAt を欠く発送済みは
	// 到達不能な状態である。
	if status == StatusShipped && paidAt == nil {
		return xerrors.Wrap(ErrInvalidStatusID, "shipped status requires paidAt")
	}
	// 配達は発送済みからのみ到達するため、shippedAt を欠く deliveredAt は到達不能な状態である。
	if deliveredAt != nil && shippedAt == nil {
		return xerrors.Wrap(ErrInvalidStatusID, "deliveredAt requires shippedAt")
	}
	// 配達済み status と deliveredAt は同時セットの不変条件を持つ。配達済みは終端状態（TerminalStatusCodes）で
	// あり、配達後に別の status へ遷移して deliveredAt だけが残ることがないため、paidAt / shippedAt のような
	// 一方向ではなくキャンセルと同じ双条件になる。
	if (status == StatusDelivered) != (deliveredAt != nil) {
		return xerrors.Wrap(ErrInvalidStatusID, "delivered status and deliveredAt must be consistent")
	}
	return nil
}

// ID は、商品 ID を返します。
func (l LockedProduct) ID() uuid.UUID { return l.id }

// Price は、単価（価格スケール・ドル decimal）を返します。
func (l LockedProduct) Price() money.Price { return l.price }

// Quantity は、ロック時点の在庫数を返します。
func (l LockedProduct) Quantity() int { return l.quantity }

// ID は、明細 ID を返します。
func (d PurchaseDetail) ID() uuid.UUID { return d.id }

// ProductID は、商品 ID を返します。
func (d PurchaseDetail) ProductID() uuid.UUID { return d.productID }

// Quantity は、購入数量を返します。
func (d PurchaseDetail) Quantity() int { return d.quantity }

// UnitPrice は、単価スナップショット（価格スケール・ドル decimal）を返します。
func (d PurchaseDetail) UnitPrice() money.Price { return d.unitPrice }

// ID は、購入 ID を返します。
func (p *Purchase) ID() uuid.UUID { return p.id }

// Code は、購入コード（UUIDv7 文字列）を返します。
func (p *Purchase) Code() string { return p.code }

// UserID は、購入したユーザーの ID を返します。
func (p *Purchase) UserID() uuid.UUID { return p.userID }

// StatusID は、購入ステータス ID を返します。New で生成した集約ではゼロ値で、再構築時に設定されます。
func (p *Purchase) StatusID() uuid.UUID { return p.statusID }

// StatusCode は、購入ステータスのコードを返します。New で生成した集約は未処理（1）、再構築時は
// 購入ステータスマスタで解決した現在状態です。Cancel 後はキャンセル（6）になります。
// Status は、購入のステータスを返します。名前・終端性・遷移可否はこの値に問い合わせます。
func (p *Purchase) Status() Status { return p.status }

// StatusCode は、永続化と外部公開に用いる業務キーを返します。
func (p *Purchase) StatusCode() int { return p.status.Code() }

// SubtotalAmount は、小計（USD セント）を返します。
func (p *Purchase) SubtotalAmount() int { return p.subtotalAmount }

// TaxAmount は、税額（USD セント）を返します。
func (p *Purchase) TaxAmount() int { return p.taxAmount }

// ShippingFee は、送料（USD セント）を返します。
func (p *Purchase) ShippingFee() int { return p.shippingFee }

// TotalAmount は、合計（USD セント）を返します。
func (p *Purchase) TotalAmount() int { return p.totalAmount }

// OrderedAt は、注文日時を返します。New で生成した集約ではゼロ値で、再構築時に設定されます。
func (p *Purchase) OrderedAt() time.Time { return p.orderedAt }

// PaidAt は、支払い日時を返します。未支払いの場合は nil です。
func (p *Purchase) PaidAt() *time.Time { return ptr.Copy(p.paidAt) }

// CanceledAt は、キャンセル日時を返します。未キャンセルの場合は nil です。
func (p *Purchase) CanceledAt() *time.Time { return ptr.Copy(p.canceledAt) }

// ShippedAt は、発送日時を返します。未発送の場合は nil です。
func (p *Purchase) ShippedAt() *time.Time { return ptr.Copy(p.shippedAt) }

// DeliveredAt は、配達日時を返します。未配達の場合は nil です。
func (p *Purchase) DeliveredAt() *time.Time { return ptr.Copy(p.deliveredAt) }

// Cancel は、購入をキャンセル状態へ遷移させます。キャンセル可能状態（未処理 / 受付中 / 確認中 / 処理中 /
// 支払い済み）からのみ遷移でき、statusCode をキャンセル（6）へ、canceledAt を now へ同時に更新します。
// 既にキャンセル済みなら ErrAlreadyCanceled、完了・配達済み・発送済み（shippedAt）なら
// ErrCancelNotAllowed をそれぞれ返します（いずれも 409）。now は時刻境界から供給します（ドメインの時刻直依存を避けるため）。
// 遷移に成功したときだけキャンセルの事実（Event）を返します。起きたことを知っているのは遷移を起こした
// この集約だけなので、事実の宣言も集約が行います。
func (p *Purchase) Cancel(now time.Time) (Event, error) {
	// canceledAt は Reconstruct の不変条件で statusCode==キャンセル と同値なので status だけで判定できる。
	if p.status == StatusCanceled {
		return Event{}, ErrAlreadyCanceled
	}
	// 配達済みも deliveredAt と同値なので status で判定する。発送済みは status が配達済みへ進んでも
	// 遷移不可であり続けるため、status ではなく残り続ける shippedAt で判定する。
	// 発送済みは status が配達済みへ進んでも遷移不可であり続けるため、残り続ける shippedAt で判定する。
	if !p.status.CanTransitionTo(StatusCanceled) || p.shippedAt != nil {
		return Event{}, ErrCancelNotAllowed
	}
	p.status = StatusCanceled
	p.canceledAt = &now
	return newEvent(EventCanceled, p.id, now), nil
}

// Pay は、購入を支払い済み状態へ遷移させます。未払い相当（未処理 / 受付中 / 確認中 / 処理中）からのみ遷移でき、
// statusCode を支払い済み（7）へ、paidAt を now へ同時に更新します。既に支払い済みなら ErrAlreadyPaid、
// キャンセル済み・完了・配達済み・発送済み（shippedAt）なら ErrPayNotAllowed をそれぞれ返します
// （いずれも 409）。決済 SDK / PSP 連携は行わず、paidAt とステータスの記録のみを担う擬似決済です
// （決済 seam の除外は nextjs-boilerplate 側の設計判断）。
// now は時刻境界から供給します（ドメインの時刻直依存を避けるため）。
// 遷移に成功したときだけ支払いの事実（Event）を返します。起きたことを知っているのは遷移を起こした
// この集約だけなので、事実の宣言も集約が行います。
func (p *Purchase) Pay(now time.Time) (Event, error) {
	if p.status == StatusPaid {
		return Event{}, ErrAlreadyPaid
	}
	// 配達済みは deliveredAt と同値なので status で判定する。発送済みは status が配達済みへ進んでも
	// 遷移不可であり続けるため、status ではなく残り続ける shippedAt で判定する。
	// 発送済みは status が配達済みへ進んでも遷移不可であり続けるため、残り続ける shippedAt で判定する。
	if !p.status.CanTransitionTo(StatusPaid) || p.shippedAt != nil {
		return Event{}, ErrPayNotAllowed
	}
	p.status = StatusPaid
	p.paidAt = &now
	return newEvent(EventPaid, p.id, now), nil
}

// Ship は、購入を発送済み状態へ遷移させます。支払い済みからのみ遷移でき、statusCode を発送済み（8）へ、
// shippedAt を now へ同時に更新します。既に発送済みなら ErrAlreadyShipped、それ以外の状態
// （未払い相当・完了・キャンセル済み・配達済み）なら ErrShipNotAllowed をそれぞれ返します（いずれも 409）。
// 配送追跡（追跡番号 / 配送業者）は扱わず、shippedAt とステータスの記録のみを担います。
// now は時刻境界から供給します（ドメインの時刻直依存を避けるため）。
// 遷移に成功したときだけ発送の事実（Event）を返します。起きたことを知っているのは遷移を起こした
// この集約だけなので、事実の宣言も集約が行います。
func (p *Purchase) Ship(now time.Time) (Event, error) {
	if p.status == StatusShipped {
		return Event{}, ErrAlreadyShipped
	}
	if !p.status.CanTransitionTo(StatusShipped) {
		return Event{}, ErrShipNotAllowed
	}
	p.status = StatusShipped
	p.shippedAt = &now
	return newEvent(EventShipped, p.id, now), nil
}

// Deliver は、購入を配達済み状態へ遷移させます。発送済みからのみ遷移でき、statusCode を配達済み（9）へ、
// deliveredAt を now へ同時に更新します。既に配達済みなら ErrAlreadyDelivered、それ以外の状態
// （未払い相当・支払い済み・完了・キャンセル済み）なら ErrDeliverNotAllowed をそれぞれ返します（いずれも 409）。
// 配達確認の証跡（署名 / 受領写真 / GPS 位置）は扱わず、deliveredAt とステータスの記録のみを担います。
// now は時刻境界から供給します（ドメインの時刻直依存を避けるため）。
// 遷移に成功したときだけ配達の事実（Event）を返します。起きたことを知っているのは遷移を起こした
// この集約だけなので、事実の宣言も集約が行います。
func (p *Purchase) Deliver(now time.Time) (Event, error) {
	if p.status == StatusDelivered {
		return Event{}, ErrAlreadyDelivered
	}
	if !p.status.CanTransitionTo(StatusDelivered) {
		return Event{}, ErrDeliverNotAllowed
	}
	p.status = StatusDelivered
	p.deliveredAt = &now
	return newEvent(EventDelivered, p.id, now), nil
}

// Details は、購入明細のスライスを返します（内部スライスの変更を防ぐためコピーを返します）。
func (p *Purchase) Details() []PurchaseDetail {
	copied := make([]PurchaseDetail, len(p.details))
	copy(copied, p.details)
	return copied
}
