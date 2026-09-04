package cart

import (
	"go-boilerplate/internal/domain/lexicon/money"
	"go-boilerplate/pkg/ptr"
)

// maxIssuesPerItem は、1 明細に同時に立ちうる issue の最大数です。
// 公開状態・在庫・価格差はそれぞれ独立に判定されるため 3 つまで重なります。
// 軸の中の値どうし（廃番と非公開、在庫切れと在庫不足）は排他なので、軸が増えない限りこの数は変わりません。
const maxIssuesPerItem = 3

const (
	// IssueNotFound は、商品が引けなかったことを表します。
	IssueNotFound Issue = "notFound"
	// IssueUnpublished は、商品が非公開であることを表します。
	IssueUnpublished Issue = "unpublished"
	// IssueDiscontinued は、商品が廃番になったことを表します。
	IssueDiscontinued Issue = "discontinued"
	// IssueOutOfStock は、在庫が無いことを表します。
	IssueOutOfStock Issue = "outOfStock"
	// IssueInsufficientStock は、在庫が要求数量に満たないことを表します。
	IssueInsufficientStock Issue = "insufficientStock"
	// IssuePriceIncreased は、最後に提示した価格より高いことを表します。
	IssuePriceIncreased Issue = "priceIncreased"
	// IssuePriceDecreased は、最後に提示した価格より安いことを表します。
	IssuePriceDecreased Issue = "priceDecreased"
)

// Issue は、明細を商品の観測値と突き合わせた結果です。
type Issue string

// ProductSnapshot は、再評価した時点で観測した商品の値です（設計意図は
// docs/spec/domain/cart.md の Overview を参照）。
type ProductSnapshot struct {
	quantity     int
	price        money.Price
	published    bool
	discontinued bool
}

// Evaluation は、明細 1 件の再評価結果です。
type Evaluation struct {
	issues            []Issue
	availableQuantity *int
}

// ProductSnapshotAttributes は、商品の観測値一式です。published と discontinued が同型のため
// 構造体で受けます（基準は docs/rules.md の Function Signature Rules）。
type ProductSnapshotAttributes struct {
	Quantity     int
	Price        money.Price
	Published    bool
	Discontinued bool
}

// NewProductSnapshot は、商品の観測値を組み立てます。
func NewProductSnapshot(attrs ProductSnapshotAttributes) ProductSnapshot {
	return ProductSnapshot{
		quantity:     attrs.Quantity,
		price:        attrs.Price,
		published:    attrs.Published,
		discontinued: attrs.Discontinued,
	}
}

// Price は、観測した単価を返します。
func (s ProductSnapshot) Price() money.Price { return s.price }

// Issues は、立った issue を返します。空なら現時点で購入可能です。
func (e Evaluation) Issues() []Issue {
	copied := make([]Issue, len(e.issues))
	copy(copied, e.issues)

	return copied
}

// AvailableQuantity は、IssueInsufficientStock のときの今買える上限を返します。それ以外は nil です。
func (e Evaluation) AvailableQuantity() *int { return ptr.Copy(e.availableQuantity) }

// hasNoIssue は、突き合わせで issue が 1 つも立たなかったかどうかを返します。
// 「購入可能」は在籍に基づく別の判定を指す語なので用いません（docs/spec/glossary.md）。
//
// ゼロ値は「まだ突き合わせていない」であって「問題が無い」ではないため false を返します。
// Evaluate は必ず非 nil の issues を返すので、nil であることがゼロ値の印になります。
// 合算に入れるかどうかがこれで決まるため、判らないものを問題無しへ倒しません。
func (e Evaluation) hasNoIssue() bool { return e.issues != nil && len(e.issues) == 0 }

// Evaluate は、明細を商品の観測値と突き合わせて再評価結果を返します。
// snapshot が nil の場合は商品を引けなかったことを表し、IssueNotFound だけが立ちます。
func (i CartItem) Evaluate(snapshot *ProductSnapshot) Evaluation {
	if snapshot == nil {
		return Evaluation{issues: []Issue{IssueNotFound}}
	}

	issues := make([]Issue, 0, maxIssuesPerItem)
	switch {
	case snapshot.discontinued:
		issues = append(issues, IssueDiscontinued)
	case !snapshot.published:
		issues = append(issues, IssueUnpublished)
	}

	var availableQuantity *int
	switch stock := snapshot.quantity; {
	case stock <= 0:
		issues = append(issues, IssueOutOfStock)
	case stock < i.quantity:
		issues = append(issues, IssueInsufficientStock)
		availableQuantity = &stock
	}

	if i.lastSeenPrice != nil {
		switch snapshot.price.Decimal().Cmp(i.lastSeenPrice.Decimal()) {
		case 1:
			issues = append(issues, IssuePriceIncreased)
		case -1:
			issues = append(issues, IssuePriceDecreased)
		}
	}

	return Evaluation{issues: issues, availableQuantity: availableQuantity}
}
