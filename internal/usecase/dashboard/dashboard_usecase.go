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
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/dashboard/query"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// resourceKindDashboard は、認可対象リソースの種別です。
const resourceKindDashboard = "dashboard"

const (
	// periodToday は、今日を集計対象とする区分です。
	periodToday = "today"
	// periodMonth は、今月を集計対象とする区分です。
	periodMonth = "month"
	// periodRange は、GetSummaryParams の From / To で指定した期間を集計対象とする区分です。
	periodRange = "range"
)

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
	GetDashboardSummary(ctx context.Context, authn *auth.Authn, params GetSummaryParams) (SummaryView, error)
}

type usecase struct {
	tracer      observability.LayerTracer
	authorizer  authz.Authorizer
	qs          query.DashboardQueryService
	productRepo product.Repository
	clk         clock.Clock
	loc         *time.Location
}

// New は、admin ダッシュボードの横断集計の参照ユースケースを生成します。
func New(
	qs query.DashboardQueryService,
	productRepo product.Repository,
	authorizer authz.Authorizer,
	clk clock.Clock,
	loc *time.Location,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer:      tf.Usecase(),
		authorizer:  authorizer,
		qs:          qs,
		productRepo: productRepo,
		clk:         clk,
		loc:         loc,
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

	// 集計対象期間はここで暦日まで確定させる。現在時刻とタイムゾーンへの依存を usecase 層に集約し、
	// クエリサービスへは解決済みの半開区間だけを渡す。
	window, err := resolveWindow(params, u.clk.Now(), u.loc)
	if err != nil {
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

// resolveWindow は、入力期間を集計対象の半開区間 [After, Before) へ解決します。
// 未知区分の扱いは GetSummaryParams.Period のドキュメントを参照。
// range のときだけ開始日・終了日の相関を検証するため、QueryService へは検証済みの区間だけが渡ります。
// 暦日の境界は loc で解釈します。loc は設定のタイムゾーンから構築された値であり、実行環境の time.Local には
// 依存しません（コンテナの既定は UTC のため、依存させると設定と異なる暦日で集計してしまいます）。
func resolveWindow(params GetSummaryParams, now time.Time, loc *time.Location) (query.Window, error) {
	switch params.Period {
	case periodMonth:
		// 呼出側の変換有無に依存せず loc 基準の暦日を得るため、ここで現在時刻を loc へ移す。
		start := startOfMonth(now.In(loc), loc)
		return query.Window{After: start, Before: start.AddDate(0, 1, 0)}, nil
	case periodRange:
		if params.From == nil || params.To == nil {
			return query.Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "period=range requires both from and to")
		}
		// From / To は利用者が指定した暦日そのものを表すため、now と違い loc へ変換してはならない
		// （UTC より西のロケーションでは前日へずれる）。
		from, to := startOfDay(*params.From, loc), startOfDay(*params.To, loc)
		if to.Before(from) {
			return query.Window{}, xerrors.Wrap(apperror.ErrInvalidArgument, "to must not be before from")
		}
		return query.Window{After: from, Before: to.AddDate(0, 0, 1)}, nil
	// 未知の period は OpenAPI の enum があるため到達しませんが、到達したときに月次や任意区間で
	// 答えるより今日で答えるほうが害が小さいため、periodToday と同じ窓へ倒します。
	case periodToday:
		fallthrough
	default:
		start := startOfDay(now.In(loc), loc)
		return query.Window{After: start, Before: start.AddDate(0, 0, 1)}, nil
	}
}

// startOfDay は、t が表す年月日の開始時刻を loc のゾーンで返します。年月日は t 自身のロケーションで解釈し、
// loc は返り値のゾーンとしてのみ用います（t を loc へ変換し直しません）。
func startOfDay(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// startOfMonth は、t が表す年月の初日の開始時刻を loc のゾーンで返します。
func startOfMonth(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
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
