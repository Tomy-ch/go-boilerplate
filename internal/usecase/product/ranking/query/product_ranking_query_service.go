//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package query は、商品売上ランキングのクエリサービスインターフェースを提供します。
package query

import (
	"context"

	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
)

const (
	// PeriodAll は、全期間を集計対象とする区分です。
	PeriodAll Period = "all"
	// Period30d は、直近30日を集計対象とする区分です。
	Period30d Period = "30d"
)

// Period は、ランキング集計の対象期間区分です。実際の境界時刻はインフラ層が現在時刻から算出します。
type Period string

// ProductRankingQueryService は、購入明細を集計した商品売上ランキングの参照を提供するクエリサービスです。
type ProductRankingQueryService interface {
	// ListRanking は、販売数量の降順（同数量は商品 ID 昇順）で上位 params.Limit 件のランキングを返します。
	ListRanking(ctx context.Context, params RankingQueryParams) ([]RankingResult, error)
}

// RankingQueryParams は、ランキング集計の入力パラメータです。
type RankingQueryParams struct {
	// Period は、集計対象期間の区分です。境界時刻の算出はインフラ層に閉じます。
	Period Period
	// Limit は、取得する上位件数です。
	Limit int
}

// RankingResult は、ランキング 1 商品分の集計結果です。
type RankingResult struct {
	ProductID    uuid.UUID
	Name         string
	Price        decimal.Decimal
	SoldQuantity int64
}
