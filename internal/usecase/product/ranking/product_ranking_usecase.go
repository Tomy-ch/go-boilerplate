//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package ranking は、商品売上ランキングの参照ユースケースを提供します。
package ranking

import (
	"context"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/ranking/query"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
)

const (
	// defaultLimit は、limit 未指定時に用いる既定の取得件数です。
	defaultLimit = 10
	// minLimit / maxLimit は、取得件数のクランプ範囲です。
	minLimit = 1
	maxLimit = 100
)

// GetRankingParams は、ランキング取得ユースケースの入力パラメータです。
type GetRankingParams struct {
	// Period は、集計対象期間（"all" / "30d"）です。未知値・空は全期間として扱います。
	Period string
	// Limit は、取得する上位件数です。0 以下は既定値、範囲外は [minLimit, maxLimit] にクランプします。
	Limit int
}

// RankingView は、商品売上ランキングのユースケース出力 DTO です。
type RankingView struct {
	// Rankings は、販売数量の降順（同数量は商品 ID 昇順）で並んだ項目一覧です。
	Rankings []RankingItemView
}

// RankingItemView は、ランキング 1 商品分の出力 DTO です。Price はサブセント精度を保持する十進量です。
type RankingItemView struct {
	ProductID    uuid.UUID
	Name         string
	Price        decimal.Decimal
	SoldQuantity int64
}

// Usecase は、商品売上ランキングの参照ユースケースを定義します。
type Usecase interface {
	// GetProductsRanking は、集計期間と件数に基づく商品売上ランキングを返します。
	GetProductsRanking(ctx context.Context, params GetRankingParams) (RankingView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	qs     query.ProductRankingQueryService
}

// New は、商品売上ランキングの参照ユースケースを生成します。
func New(qs query.ProductRankingQueryService, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		qs:     qs,
	}
}

// GetProductsRanking は、集計期間と件数を正規化し、クエリサービスの集計結果を DTO へ写像して返します。
func (u *usecase) GetProductsRanking(ctx context.Context, params GetRankingParams) (RankingView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	results, err := u.qs.ListRanking(ctx, query.RankingQueryParams{
		Period: normalizePeriod(params.Period),
		Limit:  normalizeLimit(params.Limit),
	})
	if err != nil {
		return RankingView{}, err
	}

	items := make([]RankingItemView, len(results))
	for i, r := range results {
		items[i] = RankingItemView{
			ProductID:    r.ProductID,
			Name:         r.Name,
			Price:        r.Price,
			SoldQuantity: r.SoldQuantity,
		}
	}

	return RankingView{Rankings: items}, nil
}

// normalizePeriod は、入力期間を集計区分へ正規化します。"30d" のみ直近30日、それ以外は全期間として扱います。
func normalizePeriod(period string) query.Period {
	if period == string(query.Period30d) {
		return query.Period30d
	}
	return query.PeriodAll
}

// normalizeLimit は、取得件数を既定値の適用とクランプで正規化します。
func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}
