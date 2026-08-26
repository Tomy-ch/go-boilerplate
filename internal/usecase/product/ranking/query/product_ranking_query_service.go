//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package query は、商品売上ランキングのクエリサービスインターフェースを提供します。
package query

import (
	"context"
	"time"

	"go-boilerplate/internal/usecase/tools/timewindow"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
)

// ProductRankingQueryService は、購入明細を集計した商品売上ランキングの参照を提供するクエリサービスです。
// 2 つのメソッドは母集団（公開中の商品 × キャンセルされていない購入の明細）が同一で、集計する指標と
// 並び順だけが異なります。
type ProductRankingQueryService interface {
	// ListQuantityRanking は、販売数量の降順（同数量は商品 ID 昇順）で上位 params.Limit 件を返します。
	ListQuantityRanking(ctx context.Context, params RankingQueryParams) ([]QuantityRankingResult, error)
	// ListAmountRanking は、売上金額の降順（同額は商品 ID 昇順）で上位 params.Limit 件を返します。
	// 金額は価格スケールの正確な decimal で、決済スケールへは丸めません。
	ListAmountRanking(ctx context.Context, params RankingQueryParams) ([]AmountRankingResult, error)
}

// RankingQueryParams は、ランキング集計の入力パラメータです。
type RankingQueryParams struct {
	// Window は、集計対象期間です。境界を持たない側には制限を設けません。
	Window timewindow.Window
	// Limit は、取得する上位件数です。
	Limit int
}

// QuantityRankingResult は、販売数量ランキング 1 商品分の集計結果です。
type QuantityRankingResult struct {
	ProductID uuid.UUID
	Name      string
	Price     decimal.Decimal
	// PublishedAt は、公開日時です。集計は公開中の商品に絞られており、返却行が本当にその条件を
	// 満たすかを呼び出し側がドメインの定義で確かめるために保持します。出力 DTO へは写しません。
	PublishedAt  *time.Time
	SoldQuantity int64
}

// AmountRankingResult は、売上金額ランキング 1 商品分の集計結果です。
// SalesAmount は明細の単価 × 数量の総和で、価格スケールの正確な decimal です。
type AmountRankingResult struct {
	ProductID uuid.UUID
	Name      string
	Price     decimal.Decimal
	// PublishedAt の役割は QuantityRankingResult を参照。
	PublishedAt *time.Time
	SalesAmount decimal.Decimal
}
