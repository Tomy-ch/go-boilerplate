//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package cart は、カートの参照ユースケースを提供します。主体（認証済みユーザーまたはゲストセッション）
// からカートを解決し、明細ごとに商品の現在値を突き合わせて状態を判定します。
// 単価は価格スケール（ドル decimal）、小計は決済スケール（整数セント）で扱います
// （ADR-0037 (two-scale-quantity-model)）。
package cart

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/cart"
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/boundary/token"
	"go-boilerplate/internal/usecase/boundary/tx"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// cartTTL は、Cart.Touch へ供給する有効期限の長さです（docs/spec/cart/domain.md の Touch）。
	cartTTL = 30 * 24 * time.Hour

	// ItemIssueNotFound は、商品を引けなかったことを表します。
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

// ErrUnavailableProduct は、カートへ入れられない商品を指定した場合のエラーです（422）。
// 不存在の商品と非公開の商品のどちらもこれになります（区別しない理由と、明細に立つ issue との
// 違いは docs/spec/cart/usecase.md の SetItem）。
var ErrUnavailableProduct = xerrors.Wrap(apperror.ErrValidation, "product is unavailable for the cart")

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

// SetItemParams は、明細の数量設定の入力 DTO です。
type SetItemParams struct {
	// Subject は、カートの主体です。
	Subject Subject
	// ProductID は、対象の商品です。
	ProductID uuid.UUID
	// Quantity は、設定する数量です。現在の数量への加算ではありません。
	Quantity int
}

// RemoveItemParams は、明細の削除の入力 DTO です。
type RemoveItemParams struct {
	// Subject は、カートの主体です。
	Subject Subject
	// ProductID は、取り除く明細が指す商品です。
	ProductID uuid.UUID
}

// MergeOnLoginParams は、ログイン時のカート引き継ぎの入力 DTO です。
type MergeOnLoginParams struct {
	// UserID は、引き継ぎ先の認証済みユーザーです。
	UserID uuid.UUID
	// SessionToken は、引き継ぎ元のゲストカートのトークンです。
	SessionToken string
}

// MergeCartResult は、引き継ぎで失われた分の出力 DTO です。
// 引き継ぎ後のカートの中身は持ちません（取得でいつでも引けるため）。
type MergeCartResult struct {
	// Clamped は、合算の結果が明細ごとの上限を超え、上限へ丸めた商品です。
	Clamped []uuid.UUID
	// Dropped は、明細数の上限を超えたため取り込まれなかった商品です。
	Dropped []uuid.UUID
}

// CartItemView は、明細 1 件の出力 DTO です。
type CartItemView struct {
	// ProductID は、対象の商品です。カート内で明細を一意に指す自然キーでもあります。
	ProductID uuid.UUID
	// ProductName / UnitPrice は、取得時点の商品の値です。商品を引けなかった場合は nil です。
	ProductName *string
	UnitPrice   *decimal.Decimal
	// ImagePath は、取得時点の商品の代表画像のオブジェクトキーです。
	// 商品を引けなかった場合と、商品が代表画像を持たない場合は nil です。
	ImagePath *string
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

	// SetItem は、主体のカートに指定商品の数量を設定し、再評価つきのカートを返します。
	// 現在の数量への加算ではないため、同じ入力を何回与えても結果は変わりません。
	// カートを持たない主体にはこの操作がカートを作ります（ゲストにはトークンを発行します）。
	// カートへ入れられない商品（不存在・非公開）は ErrUnavailableProduct を返します。
	SetItem(ctx context.Context, params SetItemParams) (CartView, error)

	// RemoveItem は、主体のカートから指定商品の明細を取り除きます。
	// 対象の明細が無い場合も、カートを持たない主体が呼んだ場合も成功します（削除は冪等です）。
	// カートは作らず、商品の存在も公開状態も確かめません。
	// カートを持つ主体には有効期限の延長だけが起きます。
	RemoveItem(ctx context.Context, params RemoveItemParams) error

	// ClearCart は、主体のカートから明細をすべて取り除きます。
	// カートの行は残り、有効期限は延長されます（空のカートは正当な状態です）。
	// 既に空の場合も、カートを持たない主体が呼んだ場合も成功します。カートは作りません。
	ClearCart(ctx context.Context, subject Subject) error

	// MergeOnLogin は、ゲストカートを認証済みユーザーへ引き継ぎます。
	// 引き継ぎ元を引けない場合（引き継ぎ済み・期限切れ・未知のトークン）も成功し、失われた分は空になります。
	// 形式が不正なトークンはエラーを返しますが、それ以外の理由で引き継ぎが失敗することはありません。
	// 数量は上限へ丸め、明細数の超過は古い順に残して切り捨て、失われた分を戻り値で報告します。
	MergeOnLogin(ctx context.Context, params MergeOnLoginParams) (MergeCartResult, error)
}

type usecase struct {
	tracer      observability.LayerTracer
	txm         tx.Manager
	cartRepo    cart.Repository
	productRepo product.Repository
	tokenGen    token.Generator
	clock       clock.Clock
}

// New は、カートのユースケースを生成します。
func New(
	txm tx.Manager,
	cartRepo cart.Repository,
	productRepo product.Repository,
	tokenGen token.Generator,
	clk clock.Clock,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer:      tf.Usecase(),
		txm:         txm,
		cartRepo:    cartRepo,
		productRepo: productRepo,
		tokenGen:    tokenGen,
		clock:       clk,
	}
}

// resolveCart は、主体から表示すべきカートを解決します。
// 主体を持たない・カートが無い・有効期限を過ぎているのいずれも「表示するカートが無い」であり、
// found=false を返します。いずれも呼び出し側の失敗ではありません。
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

// buildView は、明細を再評価して出力 DTO を組み立て、あわせてカートの状態を進めます。
// 提示した価格の記録と有効期限の延長までを含むため、呼び出し側は戻り値を返す前にカートを保存します。
//
// 判定も合算も提示価格を書き換える前に済ませます（docs/spec/cart/domain.md の Cart.Subtotal）。
//
// SessionToken は埋めません。トークンを新しく発行したかどうかを知るのはカートを解決した側だけです。
func (u *usecase) buildView(ctx context.Context, c *cart.Cart, now time.Time) (CartView, error) {
	items := c.Items()
	products, err := u.findProducts(ctx, items)
	if err != nil {
		return CartView{}, err
	}

	views, seen := evaluateItems(items, products)
	subtotal, serr := c.Subtotal(toSnapshots(products))
	if serr != nil {
		return CartView{}, serr
	}

	c.MarkSeen(seen)
	c.Touch(now, cartTTL)

	expiresAt := c.ExpiresAt()
	return CartView{Items: views, SubtotalAmount: subtotal, ExpiresAt: &expiresAt}, nil
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

// toSnapshots は、引けた商品を再評価用の観測値へ切り出します。
// 引けなかった商品は含めません。カートはこの表に無い明細を合算に入れません。
func toSnapshots(products map[uuid.UUID]*product.Product) map[uuid.UUID]cart.ProductSnapshot {
	snapshots := make(map[uuid.UUID]cart.ProductSnapshot, len(products))
	for id, p := range products {
		snapshots[id] = cart.NewProductSnapshot(p.Quantity(), p.Price(), p.IsPublished())
	}

	return snapshots
}

// evaluateItem は、明細 1 件の再評価をドメインへ委ね、結果を出力 DTO へ写します。
// 判定そのものは cart.CartItem.Evaluate が持ちます。この層が持つのは、商品エンティティから
// 観測値を切り出すことと、結果を DTO の語彙へ移すことだけです。
func evaluateItem(item cart.CartItem, p *product.Product) CartItemView {
	view := CartItemView{ProductID: item.ProductID(), Quantity: item.Quantity()}

	var snapshot *cart.ProductSnapshot
	if p != nil {
		name, price := p.Name(), p.Price().Decimal()
		view.ProductName, view.UnitPrice = &name, &price
		if image, ok := p.PrimaryImage(); ok {
			imagePath := image.ImagePath()
			view.ImagePath = &imagePath
		}
		s := cart.NewProductSnapshot(p.Quantity(), p.Price(), p.IsPublished())
		snapshot = &s
	}

	evaluation := item.Evaluate(snapshot)
	view.Issues = toItemIssues(evaluation.Issues())
	view.AvailableQuantity = evaluation.AvailableQuantity()

	return view
}

// toItemIssues は、ドメインの再評価結果を出力 DTO の語彙へ写します。
func toItemIssues(issues []cart.Issue) []ItemIssue {
	converted := make([]ItemIssue, len(issues))
	for i, issue := range issues {
		converted[i] = toItemIssue(issue)
	}
	return converted
}

// toItemIssue は、再評価結果 1 件を出力 DTO の語彙へ写します。
// 対応を持たない値は写像できないため、黙って既定値へ倒さず panic で異常を知らせます。
func toItemIssue(issue cart.Issue) ItemIssue {
	switch issue {
	case cart.IssueNotFound:
		return ItemIssueNotFound
	case cart.IssueUnpublished:
		return ItemIssueUnpublished
	case cart.IssueOutOfStock:
		return ItemIssueOutOfStock
	case cart.IssueInsufficientStock:
		return ItemIssueInsufficientStock
	case cart.IssuePriceIncreased:
		return ItemIssuePriceIncreased
	case cart.IssuePriceDecreased:
		return ItemIssuePriceDecreased
	default:
		panic(fmt.Sprintf("cart: unknown item issue: %q", issue))
	}
}

// emptyCartView は、表示するカートが無いときの応答です。
// 明細は空スライスで、有効期限は行が存在しないため nil です。
func emptyCartView() CartView {
	return CartView{Items: []CartItemView{}}
}
