//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package query

import (
	"context"

	"go-boilerplate/internal/usecase/purchase/period"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
)

// PurchaseSummaryQueryService は、購入の集計 read 投影を提供する QueryService です。
// 件数・合計金額・ステータス別内訳は購入集約を再構成できない派生投影であるため、Repository ではなく
// 読み取り側に置きます（ADR-0032 (lightweight-cqrs)）。商品単位の集計は購入・商品・商品カテゴリを
// またぐ結合形そのものを読むため、同じく読み取り側です。
// いずれのメソッドも母集団は同一（所有権・キャンセル除外・window）で、絞り込まない window を渡した
// 場合は全期間を対象とします。対象が存在しない場合はいずれも空値（スライス / ゼロ値）を返し、エラーとしません。
type PurchaseSummaryQueryService interface {
	// SummarizeByUserID は、認証主体（userID）の購入をステータス単位に集計し、購入ステータスマスタの表示順で返します。
	// 母集団・対象なし時の扱いは PurchaseSummaryQueryService の doc コメントを参照。
	SummarizeByUserID(ctx context.Context, userID uuid.UUID, window period.Window) ([]PurchaseStatusSummaryReadModel, error)
	// SumItemsByUserID は、認証主体（userID）の購入明細の金額合計（単価 × 数量の総和）を返します。
	// 価格スケールの正確な decimal で、決済スケール（セント整数）へは丸めません（ADR-0038 (two-scale-quantity-model)）。
	// 対象なし時の扱いは PurchaseSummaryQueryService の doc コメントを参照。
	SumItemsByUserID(ctx context.Context, userID uuid.UUID, window period.Window) (decimal.Decimal, error)
	// SummarizeItemsByProductByUserID は、認証主体（userID）の購入明細を商品単位に集計し、商品が属する
	// カテゴリを添えて返します。カテゴリ単位の集計は呼び出し側がこの結果を畳み込んで得ます。
	// 対象なし時の扱いは PurchaseSummaryQueryService の doc コメントを参照。
	SummarizeItemsByProductByUserID(ctx context.Context, userID uuid.UUID, window period.Window) ([]PurchaseItemSummaryReadModel, error)
}

// PurchaseStatusSummaryReadModel は、1 ステータス分の購入集計の読み取りモデルです。
// 金額は決済スケールの整数（USD セント）です。
type PurchaseStatusSummaryReadModel struct {
	// StatusID / StatusName は、購入ステータスマスタで解決済みの ID と名称です。
	// StatusCode は、購入ステータスの業務キー（Status.Code）です。
	StatusID   uuid.UUID
	StatusCode int
	StatusName string
	// Count は、当該ステータスの購入件数です。
	Count int64
	// TotalAmount は、当該ステータスの購入金額の合計です。
	TotalAmount int64
}

// PurchaseItemSummaryReadModel は、1 商品分の購入明細集計の読み取りモデルです。
// ItemsTotal は価格スケールの正確な decimal（USD ドル）です。
type PurchaseItemSummaryReadModel struct {
	// CategoryName は、商品が属する商品カテゴリの名称です。
	CategoryName string
	// ProductID / ProductName は、商品の ID と名称です。名称は一意ではないため、識別には ID を用います。
	ProductID   uuid.UUID
	ProductName string
	// ItemsTotal は、当該商品の明細金額の合計（単価 × 数量の総和）です。
	ItemsTotal decimal.Decimal
}
