//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package dashboard は、admin ダッシュボードの横断集計の参照ユースケースを提供します。
package dashboard

import (
	"context"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/dashboard/query"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// resourceKindDashboard は、認可対象リソースの種別です。
const resourceKindDashboard = "dashboard"

// GetSummaryParams は、ダッシュボード集計取得ユースケースの入力パラメータです。
type GetSummaryParams struct {
	// Period は、集計対象期間の区分（"today" / "month" / "range"）です。未知値・空は today として扱います。
	Period string
	// From / To は、Period が "range" のときに必須の開始日・終了日です（両端を含みます）。
	// 暦日としての年月日のみが意味を持ちます。他の区分では参照しません。
	From *time.Time
	To   *time.Time
}

// StatusCountView は、購入ステータス別件数 1 件分のユースケース出力 DTO です。
// ステータスは購入ステータスマスタで解決済みの ID と名称です。
type StatusCountView struct {
	StatusID   uuid.UUID
	StatusName string
	Count      int64
}

// SummaryView は、ダッシュボード横断集計のユースケース出力 DTO です。SalesAmount は USD セント単位の整数です。
type SummaryView struct {
	// SalesAmount / SalesCount は、集計対象期間の売上合計と、その母集団の購入件数です。
	// キャンセル済みの購入は除外し、未払いの購入は含みます。対象がない場合はいずれも 0 です。
	SalesAmount int64
	SalesCount  int64
	// PurchaseStatusCounts は、集計対象期間の購入のステータス別件数です。売上と異なりキャンセル済みも含みます。
	// 期間内に購入が出現したステータスのみを含むため、購入がない場合は空スライスです。
	PurchaseStatusCounts []StatusCountView
	// TotalProductCount / PublishedProductCount は、登録商品の総数と公開済み件数です。集計対象期間に依存しません。
	TotalProductCount     int64
	PublishedProductCount int64
}

// Usecase は、admin ダッシュボードの横断集計の参照ユースケースを定義します。
type Usecase interface {
	// GetDashboardSummary は、集計対象期間の売上・購入ステータス別件数と、商品数を 1 つの DTO に合成して返します。
	// 管理者以外は拒否します。対象データがない場合はゼロ値を返します。
	GetDashboardSummary(ctx context.Context, authn *auth.Authn, params GetSummaryParams) (SummaryView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer      observability.LayerTracer
	authorizer  authz.Authorizer
	qs          query.DashboardQueryService
	productRepo product.Repository
}

// New は、admin ダッシュボードの横断集計の参照ユースケースを生成します。
func New(
	qs query.DashboardQueryService,
	productRepo product.Repository,
	authorizer authz.Authorizer,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer:      tf.Usecase(),
		authorizer:  authorizer,
		qs:          qs,
		productRepo: productRepo,
	}
}

func (u *usecase) GetDashboardSummary(
	ctx context.Context, authn *auth.Authn, params GetSummaryParams,
) (SummaryView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return SummaryView{}, xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")
	}
	// ダッシュボードは運用（admin）の参照で所有者概念を持たないため、所有者なしリソースとして認可する。
	// 所有者を渡さないことで Authorizer の所有者フォールバックが働かず、admin だけが通る。
	if err := u.authorizer.Authorize(
		ctx, authn, authz.ActionDashboardRead, authz.NewResource(resourceKindDashboard, nil),
	); err != nil {
		return SummaryView{}, err
	}

	period, err := normalizePeriod(params)
	if err != nil {
		return SummaryView{}, err
	}

	sales, err := u.qs.SummarizeSales(ctx, period)
	if err != nil {
		return SummaryView{}, err
	}
	statusCounts, err := u.qs.CountPurchasesByStatus(ctx, period)
	if err != nil {
		return SummaryView{}, err
	}
	// 商品数は商品集約の属性による単純な件数であり、期間集計の投影ではないため Repository から取る。
	productCounts, err := u.productRepo.Count(ctx)
	if err != nil {
		return SummaryView{}, err
	}

	return SummaryView{
		SalesAmount:           sales.Amount,
		SalesCount:            sales.Count,
		PurchaseStatusCounts:  toStatusCountViews(statusCounts),
		TotalProductCount:     productCounts.Total,
		PublishedProductCount: productCounts.Published,
	}, nil
}

// normalizePeriod は、入力期間を集計区分へ正規化します。"month" / "range" のみ該当区分とし、それ以外は today として扱います。
// range のときだけ開始日・終了日の相関を検証し、QueryService へは検証済みの指定だけが渡ります。
func normalizePeriod(params GetSummaryParams) (query.Period, error) {
	switch params.Period {
	case string(query.PeriodMonth):
		return query.Period{Kind: query.PeriodMonth}, nil
	case string(query.PeriodRange):
		if params.From == nil || params.To == nil {
			return query.Period{}, xerrors.Wrap(apperror.ErrInvalidArgument, "period=range requires both from and to")
		}
		// 開始日・終了日は暦日のみが意味を持つため、比較も QueryService へ渡す値も暦日へ揃える。
		from, to := dateOnly(*params.From), dateOnly(*params.To)
		if to.Before(from) {
			return query.Period{}, xerrors.Wrap(apperror.ErrInvalidArgument, "to must not be before from")
		}
		return query.Period{Kind: query.PeriodRange, From: from, To: to}, nil
	default:
		return query.Period{Kind: query.PeriodToday}, nil
	}
}

// dateOnly は、時刻成分を落として t の暦日だけを表す時刻を返します。ロケーションは t のものを保ちます。
func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// toStatusCountViews は、ステータス別件数の集計結果を出力 DTO へ写像します。
func toStatusCountViews(results []query.PurchaseStatusCountResult) []StatusCountView {
	views := make([]StatusCountView, len(results))
	for i, r := range results {
		views[i] = StatusCountView{
			StatusID:   r.StatusID,
			StatusName: r.StatusName,
			Count:      r.Count,
		}
	}
	return views
}
