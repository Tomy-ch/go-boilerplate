//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package dashboard は、admin ダッシュボードの横断集計の参照ユースケースを提供します。
package dashboard

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/domain/product"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/authz"
	"go-boilerplate/internal/usecase/dashboard/query"
	"go-boilerplate/internal/usecase/tools/timewindow"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// resourceKindDashboard は、認可対象リソースの種別です。
const resourceKindDashboard = "dashboard"

// StatusCountView は、購入ステータス別件数 1 件分のユースケース出力 DTO です。
// ステータスは購入ステータスマスタで解決済みの ID・業務キー・名称です。
type StatusCountView struct {
	StatusID   uuid.UUID
	StatusCode int
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
	GetDashboardSummary(ctx context.Context, authn *auth.Authn, window timewindow.Window) (SummaryView, error)
}

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
	ctx context.Context, authn *auth.Authn, window timewindow.Window,
) (SummaryView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return SummaryView{}, xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")
	}
	// 所有者なしリソースとして認可する（docs/spec/usecase/dashboard.md 参照）。
	if err := u.authorizer.Authorize(
		ctx, authn, authz.ActionDashboardRead, authz.NewResource(resourceKindDashboard, nil),
	); err != nil {
		return SummaryView{}, err
	}

	sales, err := u.qs.SummarizeSales(ctx, window)
	if err != nil {
		return SummaryView{}, err
	}
	statusCounts, err := u.qs.CountPurchasesByStatus(ctx, window)
	if err != nil {
		return SummaryView{}, err
	}
	// 商品数は Repository から取得する（配置根拠は docs/spec/usecase/dashboard.md 参照）。
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

// toStatusCountViews は、ステータス別件数の集計結果を出力 DTO へ写像します。
func toStatusCountViews(results []query.PurchaseStatusCountResult) []StatusCountView {
	views := make([]StatusCountView, len(results))
	for i, r := range results {
		views[i] = StatusCountView{
			StatusID:   r.StatusID,
			StatusCode: r.StatusCode,
			StatusName: r.StatusName,
			Count:      r.Count,
		}
	}
	return views
}
