//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package ranking は、商品売上ランキングの参照ユースケースを提供します。
package ranking

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/product/ranking/query"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/internal/usecase/tools/timewindow"
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
	// Window は、集計対象期間です。ゼロ値は全期間を意味します。
	Window timewindow.Window
	// Limit は、取得する上位件数です。0 以下は既定値 10 を適用し、100 を超える値は 100 にクランプします。
	Limit int
}

// QuantityRankingView は、販売数量ランキングのユースケース出力 DTO です。
type QuantityRankingView struct {
	// Rankings は、販売数量の降順（同数量は商品 ID 昇順）で並んだ項目一覧です。
	Rankings []QuantityRankingItemView
}

// QuantityRankingItemView は、販売数量ランキング 1 商品分の出力 DTO です。
// Price はサブセント精度を保持する十進量です。
type QuantityRankingItemView struct {
	ProductID    uuid.UUID
	Name         string
	Price        decimal.Decimal
	SoldQuantity int64
}

// AmountRankingView は、売上金額ランキングのユースケース出力 DTO です。
type AmountRankingView struct {
	// Rankings は、売上金額の降順（同額は商品 ID 昇順）で並んだ項目一覧です。
	Rankings []AmountRankingItemView
}

// AmountRankingItemView は、売上金額ランキング 1 商品分の出力 DTO です。
// SalesAmount は明細金額の合計で、価格スケールの正確な十進量です（決済スケールへ丸めません）。
type AmountRankingItemView struct {
	ProductID   uuid.UUID
	Name        string
	Price       decimal.Decimal
	SalesAmount decimal.Decimal
}

// Usecase は、商品売上ランキングの参照ユースケースを定義します。
// 2 つのメソッドは母集団が同一で、集計する指標と並び順だけが異なります。
type Usecase interface {
	// GetQuantityRanking は、集計期間と件数に基づく販売数量ランキングを返します。
	GetQuantityRanking(ctx context.Context, params GetRankingParams) (QuantityRankingView, error)
	// GetAmountRanking は、集計期間と件数に基づく売上金額ランキングを返します。
	GetAmountRanking(ctx context.Context, params GetRankingParams) (AmountRankingView, error)
}

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

// GetQuantityRanking は、件数の正規化を usecase 側の入力方針として引き受けます。QueryService へは
// 正規化済みの値だけが渡るため、範囲外の件数が集計側へ到達する経路はありません。
func (u *usecase) GetQuantityRanking(ctx context.Context, params GetRankingParams) (QuantityRankingView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	results, err := u.qs.ListQuantityRanking(ctx, toQueryParams(params))
	if err != nil {
		return QuantityRankingView{}, err
	}

	for _, r := range results {
		if verr := ensurePublished(r.ProductID, r.PublishedAt); verr != nil {
			return QuantityRankingView{}, verr
		}
	}

	items := make([]QuantityRankingItemView, len(results))
	for i, r := range results {
		items[i] = QuantityRankingItemView{
			ProductID:    r.ProductID,
			Name:         r.Name,
			Price:        r.Price,
			SoldQuantity: r.SoldQuantity,
		}
	}

	return QuantityRankingView{Rankings: items}, nil
}

// GetAmountRanking は、件数の正規化について GetQuantityRanking と同じ方針を取ります。
func (u *usecase) GetAmountRanking(ctx context.Context, params GetRankingParams) (AmountRankingView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	results, err := u.qs.ListAmountRanking(ctx, toQueryParams(params))
	if err != nil {
		return AmountRankingView{}, err
	}

	for _, r := range results {
		if verr := ensurePublished(r.ProductID, r.PublishedAt); verr != nil {
			return AmountRankingView{}, verr
		}
	}

	items := make([]AmountRankingItemView, len(results))
	for i, r := range results {
		items[i] = AmountRankingItemView{
			ProductID:   r.ProductID,
			Name:        r.Name,
			Price:       r.Price,
			SalesAmount: r.SalesAmount,
		}
	}

	return AmountRankingView{Rankings: items}, nil
}

// toQueryParams は、入力を正規化して QueryService の取得条件へ変換します。
func toQueryParams(params GetRankingParams) query.RankingQueryParams {
	return query.RankingQueryParams{
		Window: params.Window,
		Limit:  paging.NewLimit(ptr.To(params.Limit), rankingLimitPolicy).Value(),
	}
}

// ensurePublished は、集計結果の 1 行が product.IsPublished を満たすことを確かめます。
// 集約を再構築しない経路のため突き合わせるのは公開日時のみです。乖離時の扱いは README の
// Verifying infrastructure against the domain を参照。
func ensurePublished(productID uuid.UUID, publishedAt *time.Time) error {
	if !product.IsPublished(publishedAt) {
		return xerrors.Wrap(errUnpublishedInRanking, productID.String())
	}

	return nil
}
