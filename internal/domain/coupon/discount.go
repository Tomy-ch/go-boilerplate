package coupon

import (
	"fmt"

	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/xerrors"
)

// 既知の値引き種別の業務キー（大小は優劣を意味しません）。
const (
	discountKindFlat = 1
	discountKindRate = 2
)

// 既知の値引き種別。
var (
	// DiscountKindFlat は、金額を直接差し引く「定額」です。
	DiscountKindFlat = DiscountKind{code: discountKindFlat, name: "flat"}
	// DiscountKindRate は、対象額に率を掛けて差し引く「定率」です。
	DiscountKindRate = DiscountKind{code: discountKindRate, name: "rate"}
)

// DiscountKind は、値引きの決まり方を表す値オブジェクトです。
//
// 内側に持つ code は永続化と外部公開のための業務キーであり、種別の間に順序はありません。
// マスタ表を持たないのは、種別が業務の語彙であってデータではないためです
// （purchase.Status と同じ理由。docs/spec/domain/coupon.md の Overview）。
type DiscountKind struct {
	code int
	name string
}

// allDiscountKinds は、既知の値引き種別一覧です。code からの解決に用います。
func allDiscountKinds() []DiscountKind {
	return []DiscountKind{DiscountKindFlat, DiscountKindRate}
}

// NewDiscountKind は、永続化されている code から値引き種別を解決します。
// 既知でない code は ErrInvalidDiscountKind を返します（永続化状態の破損を再構築時に弾くため）。
func NewDiscountKind(code int) (DiscountKind, error) {
	for _, k := range allDiscountKinds() {
		if k.code == code {
			return k, nil
		}
	}
	return DiscountKind{}, xerrors.Wrap(ErrInvalidDiscountKind, fmt.Sprintf("unknown discount kind code: %d", code))
}

// Code は、永続化と外部公開に用いる業務キーを返します。
func (k DiscountKind) Code() int { return k.code }

// Name は、値引き種別の名前を返します。外部へ種別を伝えるときは code ではなくこちらを用います。
func (k DiscountKind) Name() string { return k.name }

// IsZero は、未設定の値引き種別かどうかを返します。
func (k DiscountKind) IsZero() bool { return k.code == 0 }

// Discount は、値引きを表す値オブジェクトです。決まり方（種別）と、その種別における値の組です。
//
// 定額では value が差し引く金額（USD ドルの十進量）、定率では value が対象額に掛ける率です。
// 適用範囲は関知しません。どの明細が対象かは [Scope] が答えます。
type Discount struct {
	kind  DiscountKind
	value decimal.Decimal
}

// NewFlatDiscount は、定額の値引きを生成します。amount は正の十進量である必要があります。
// 0 以下は値引きにならないため ErrInvalidDiscountValue を返します。
func NewFlatDiscount(amount decimal.Decimal) (Discount, error) {
	if amount.Sign() <= 0 {
		return Discount{}, xerrors.Wrap(ErrInvalidDiscountValue, "flat discount amount must be positive")
	}
	return Discount{kind: DiscountKindFlat, value: amount}, nil
}

// NewRateDiscount は、定率の値引きを生成します。rate は 0 より大きく 1 以下である必要があります。
// 範囲外は ErrInvalidDiscountValue を返します。1 を超える率は対象額より多く差し引くことになり、
// 値引きの意味を失うため許しません。
func NewRateDiscount(rate decimal.Decimal) (Discount, error) {
	if rate.Sign() <= 0 || rate.Cmp(decimal.FromInt(1)) > 0 {
		return Discount{}, xerrors.Wrap(ErrInvalidDiscountValue, "rate discount must be within (0, 1]")
	}
	return Discount{kind: DiscountKindRate, value: rate}, nil
}

// ReconstructDiscount は、永続化されている種別と値から値引きを再構築します。
// 検証は生成時と同一です。
func ReconstructDiscount(kind DiscountKind, value decimal.Decimal) (Discount, error) {
	switch kind {
	case DiscountKindFlat:
		return NewFlatDiscount(value)
	case DiscountKindRate:
		return NewRateDiscount(value)
	default:
		return Discount{}, xerrors.Wrap(ErrInvalidDiscountKind, "discount kind is required")
	}
}

// Kind は、値引きの決まり方を返します。
func (d Discount) Kind() DiscountKind { return d.kind }

// Value は、種別における値を返します。定額なら差し引く金額、定率なら掛ける率です。
func (d Discount) Value() decimal.Decimal { return d.value }

// IsZero は、未設定の値引きかどうかを返します。
func (d Discount) IsZero() bool { return d.kind.IsZero() }

// Apply は、対象額に対して差し引く額を十進量で返します。
//
// 最小単位への丸めは行いません。丸めは決済額を確定する 1 箇所（money.Price.ToMinorUnit）だけで
// 起きるべきものであり、ここで丸めると同じ量が 2 度丸められます（ADR-0038）。
//
// 差し引く額が対象額を超える場合は対象額でクランプします。値引きが請求額を負にすることはないためで、
// 定額の値引きが対象額を上回るときにこれが効きます。
func (d Discount) Apply(subtotal money.Price) decimal.Decimal {
	base := subtotal.Decimal()

	var discounted decimal.Decimal
	switch d.kind {
	case DiscountKindFlat:
		discounted = d.value
	case DiscountKindRate:
		discounted = base.Mul(d.value)
	default:
		return decimal.FromInt(0)
	}

	if discounted.Cmp(base) > 0 {
		return base
	}
	return discounted
}
