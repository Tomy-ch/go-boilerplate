//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package ranking は、商品売上ランキングの参照ユースケースを提供します。
package ranking

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/ranking/query"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// defaultLimit は、limit 未指定時に用いる既定の取得件数です。
	defaultLimit = 10
	// maxLimit は、許容する最大の取得件数です。
	maxLimit = 100
)

// rankingLimitPolicy は、商品売上ランキングの取得件数規約です。OpenAPI の limit（既定 10 / 1〜100）と対応します。
var rankingLimitPolicy = paging.LimitPolicy{Default: defaultLimit, Max: maxLimit}

// errUnpublishedInRanking は、公開中の商品に絞られているはずのランキングに非公開商品が現れた場合のエラーです。
var errUnpublishedInRanking = xerrors.Wrap(apperror.ErrInternal, "unpublished product in ranking")

// GetRankingParams は、ランキング取得ユースケースの入力パラメータです。
type GetRankingParams struct {
	// Period は、集計対象期間（"all" / "30d"）です。未知値・空は全期間として扱います。
	Period string
	// Limit は、取得する上位件数です。0 以下は既定値 10 を適用し、100 を超える値は 100 にクランプします。
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

// GetProductsRanking は、集計期間と件数の正規化を usecase 側の入力方針として引き受けます。QueryService へは
// 正規化済みの値だけが渡るため、未知の期間や範囲外の件数が集計側へ到達する経路はありません。
func (u *usecase) GetProductsRanking(ctx context.Context, params GetRankingParams) (RankingView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	results, err := u.qs.ListRanking(ctx, query.RankingQueryParams{
		Period: normalizePeriod(params.Period),
		Limit:  paging.NewLimit(ptr.To(params.Limit), rankingLimitPolicy).Value(),
	})
	if err != nil {
		return RankingView{}, err
	}

	if verr := ensurePublished(results); verr != nil {
		return RankingView{}, verr
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

// ensurePublished は、集計結果の各行が product.IsPublished を満たすことを確かめます。
// 集約を再構築しない経路のため突き合わせるのは公開日時のみです。乖離時の扱いは README の
// Verifying infrastructure against the domain を参照。
func ensurePublished(results []query.RankingResult) error {
	for _, r := range results {
		if !product.IsPublished(r.PublishedAt) {
			return xerrors.Wrap(errUnpublishedInRanking, r.ProductID.String())
		}
	}

	return nil
}

// normalizePeriod は、入力期間を集計区分へ正規化します。"30d" のみ直近30日、それ以外は全期間として扱います。
func normalizePeriod(period string) query.Period {
	if period == string(query.Period30d) {
		return query.Period30d
	}
	return query.PeriodAll
}
