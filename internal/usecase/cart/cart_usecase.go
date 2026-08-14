//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package cart は、カートの参照ユースケースを提供します。主体（認証済みユーザーまたはゲストセッション）
// からカートを解決し、明細ごとに商品の現在値を突き合わせて状態を判定します。
// 単価は価格スケール（ドル decimal）、小計は決済スケール（整数セント）で扱います
// （ADR-0036 (two-scale-quantity-model)）。
package cart

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// cartTTL は、カートの有効期限の長さです。参照・更新のたびに現在時刻からこの長さだけ先へ延長されます。
	// ドメインは設定値を持たないため、この層が供給します（docs/spec/cart/domain.md の Touch）。
	cartTTL = 30 * 24 * time.Hour

	// subtotalMinorUnitDigits は、小計を決済スケール（USD セント）へ落とすときの小数桁数です。
	subtotalMinorUnitDigits = 2

	// maxIssuesPerItem は、1 明細に同時に立ちうる issue の最大数です。
	// 非公開・在庫・価格差はそれぞれ独立に判定されるため 3 つまで重なります。
	maxIssuesPerItem = 3

	// ItemIssueNotFound は、商品を引けなかったことを表します。在庫も価格も判定材料が無いため、
	// この issue は単独で立ちます。
	ItemIssueNotFound ItemIssue = "notFound"
	// ItemIssueUnpublished は、商品が非公開であることを表します。
	ItemIssueUnpublished ItemIssue = "unpublished"
	// ItemIssueOutOfStock は、在庫が無いことを表します。
	ItemIssueOutOfStock ItemIssue = "outOfStock"
	// ItemIssueInsufficientStock は、在庫が要求数量に満たないことを表します。
	ItemIssueInsufficientStock ItemIssue = "insufficientStock"
	// ItemIssuePriceIncreased は、前回提示した価格より高いことを表します。
	ItemIssuePriceIncreased ItemIssue = "priceIncreased"
	// ItemIssuePriceDecreased は、前回提示した価格より安いことを表します。
	ItemIssuePriceDecreased ItemIssue = "priceDecreased"
)

// ItemIssue は、明細ごとの再評価結果です。値は外部向けの安定コードで、表示ではなく分岐に用います。
type ItemIssue string

// Subject は、カートの主体です。認証済みユーザーとゲストセッションのうち高々一方が設定され、
// どちらも設定されていない場合は表示するカートを持ちません。
type Subject struct {
	// UserID は、認証済みの内部ユーザー ID です。
	UserID *uuid.UUID
	// SessionToken は、ゲスト追跡用のトークンです。
	SessionToken *string
}

// CartItemView は、明細 1 件の出力 DTO です。
type CartItemView struct {
	// ProductID は、対象の商品です。カート内で明細を一意に指す自然キーでもあります。
	ProductID uuid.UUID
	// ProductName / UnitPrice は、取得時点の商品の値です。商品を引けなかった場合は nil です。
	ProductName *string
	UnitPrice   *decimal.Decimal
	// Quantity は、カートに入っている数量です。
	Quantity int
	// Issues は、この明細の再評価結果です。空なら現時点で購入可能です。
	Issues []ItemIssue
	// AvailableQuantity は、ItemIssueInsufficientStock のときの今買える上限です。それ以外は nil です。
	AvailableQuantity *int
}

// CartView は、カートの出力 DTO です。
type CartView struct {
	// SessionToken は、ゲストカートのトークンです。新規発行が起きなかった操作では nil です。
	SessionToken *string
	// Items は、明細です。カートが無い場合も空スライスで、nil にはなりません。
	Items []CartItemView
	// SubtotalAmount は、購入可能な明細のみを合算した参考値（USD セント）です。請求額ではありません。
	SubtotalAmount int64
	// ExpiresAt は、カートの有効期限です。カートが存在しない場合は nil です。
	ExpiresAt *time.Time
}

// Usecase は、カートのユースケースを定義します。
type Usecase interface {
	// GetCart は、主体のカートを明細ごとの再評価つきで返します。
	// カートが無い、または有効期限を過ぎている場合は空のカートを返し、行は作りません。
	// 明細の問題（在庫・非公開・価格差）はエラーではなく、各明細の Issues として返されます。
	GetCart(ctx context.Context, subject Subject) (CartView, error)
}

type usecase struct {
	tracer      observability.LayerTracer
	txm         tx.Manager
	cartRepo    cart.Repository
	productRepo product.Repository
	clock       clock.Clock
}

// New は、カートのユースケースを生成します。
func New(
	txm tx.Manager,
	cartRepo cart.Repository,
	productRepo product.Repository,
	clk clock.Clock,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer:      tf.Usecase(),
		txm:         txm,
		cartRepo:    cartRepo,
		productRepo: productRepo,
		clock:       clk,
	}
}

// resolveCart は、主体から表示すべきカートを解決します。
// 主体を持たない・カートが無い・有効期限を過ぎているのいずれも「表示するカートが無い」であり、
// found=false を返します。いずれも失敗ではないため、呼び出し側は空のカートを返して行を作りません。
func (u *usecase) resolveCart(ctx context.Context, subject Subject, now time.Time) (*cart.Cart, bool, error) {
	c, found, err := u.findCart(ctx, subject)
	if err != nil {
		if xerrors.Is(err, apperror.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if !found || c.IsExpired(now) {
		return nil, false, nil
	}
	return c, true, nil
}

// findCart は、主体の種類に応じてカートを引きます。主体を持たない場合は found=false を返します。
// 認証済みユーザーはゲストセッションより優先されます。
func (u *usecase) findCart(ctx context.Context, subject Subject) (*cart.Cart, bool, error) {
	switch {
	case subject.UserID != nil:
		c, err := u.cartRepo.FindByOwnerID(ctx, *subject.UserID)
		return c, err == nil, err
	case subject.SessionToken != nil:
		token, err := cart.NewSessionToken(*subject.SessionToken)
		if err != nil {
			return nil, false, err
		}
		c, err := u.cartRepo.FindBySessionToken(ctx, token)
		return c, err == nil, err
	default:
		return nil, false, nil
	}
}

// findProducts は、明細が指す商品を ID で引き当てた表を返します。
// 引けなかった商品は表に現れません。明細が無い場合は問い合わせません。
func (u *usecase) findProducts(
	ctx context.Context, items []cart.CartItem,
) (map[uuid.UUID]*product.Product, error) {
	if len(items) == 0 {
		return map[uuid.UUID]*product.Product{}, nil
	}

	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProductID())
	}

	products, err := u.productRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	found := make(map[uuid.UUID]*product.Product, len(products))
	for _, p := range products {
		found[p.ID()] = p
	}
	return found, nil
}

// evaluateItems は、明細を商品の現在値と突き合わせ、明細ごとの View と、提示した価格の表を返します。
// 価格の表には引けた商品のぶんだけが入ります（引けなかった明細の提示価格は据え置かれます）。
func evaluateItems(
	items []cart.CartItem, products map[uuid.UUID]*product.Product,
) ([]CartItemView, map[uuid.UUID]money.Price) {
	views := make([]CartItemView, 0, len(items))
	seen := make(map[uuid.UUID]money.Price, len(items))

	for _, item := range items {
		p := products[item.ProductID()]
		views = append(views, evaluateItem(item, p))
		if p != nil {
			seen[p.ID()] = p.Price()
		}
	}
	return views, seen
}

// evaluateItem は、明細 1 件を商品の現在値と突き合わせます。
// 商品を引けなかった場合は notFound だけを立てます。それ以外では、公開状態・在庫・価格差を
// それぞれ独立に判定し、成立したものを併記します。在庫 0 は「不足」ではなく「無い」であるため、
// outOfStock と insufficientStock は同時に立ちません。
func evaluateItem(item cart.CartItem, p *product.Product) CartItemView {
	view := CartItemView{
		ProductID: item.ProductID(),
		Quantity:  item.Quantity(),
		Issues:    make([]ItemIssue, 0, maxIssuesPerItem),
	}

	if p == nil {
		view.Issues = append(view.Issues, ItemIssueNotFound)
		return view
	}

	name, price := p.Name(), p.Price().Decimal()
	view.ProductName, view.UnitPrice = &name, &price

	if !p.IsPublished() {
		view.Issues = append(view.Issues, ItemIssueUnpublished)
	}

	switch stock := p.Quantity(); {
	case stock <= 0:
		view.Issues = append(view.Issues, ItemIssueOutOfStock)
	case stock < item.Quantity():
		view.Issues = append(view.Issues, ItemIssueInsufficientStock)
		view.AvailableQuantity = &stock
	}

	if lastSeen := item.LastSeenPrice(); lastSeen != nil {
		switch price.Cmp(lastSeen.Decimal()) {
		case 1:
			view.Issues = append(view.Issues, ItemIssuePriceIncreased)
		case -1:
			view.Issues = append(view.Issues, ItemIssuePriceDecreased)
		}
	}

	return view
}

// subtotalAmount は、購入可能な明細だけを合算して決済スケールの整数へ落とします。
// 丸めは合算の後に一度だけ行い、明細ごとの丸め誤差が積み上がらないようにします。
func subtotalAmount(views []CartItemView) (int64, error) {
	sum := decimal.FromInt(0)
	for _, v := range views {
		if len(v.Issues) > 0 || v.UnitPrice == nil {
			continue
		}
		sum = sum.Add(v.UnitPrice.Mul(decimal.FromInt(int64(v.Quantity))))
	}
	return sum.ToScaledInt64(subtotalMinorUnitDigits)
}

// emptyCartView は、表示するカートが無いときの応答です。
// 明細は空スライスで、有効期限は行が存在しないため nil です。
func emptyCartView() CartView {
	return CartView{Items: []CartItemView{}}
}
