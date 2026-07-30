//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package query は、admin ダッシュボードの横断集計のクエリサービスインターフェースを提供します。
package query

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

const (
	// PeriodToday は、今日を集計対象とする区分です。
	PeriodToday PeriodKind = "today"
	// PeriodMonth は、今月を集計対象とする区分です。
	PeriodMonth PeriodKind = "month"
	// PeriodRange は、Period の From / To で指定した期間を集計対象とする区分です。
	PeriodRange PeriodKind = "range"
)

// PeriodKind は、集計対象期間の区分です。
type PeriodKind string

// Period は、集計対象期間の指定です。today / month の境界はインフラ層が現在時刻から算出し、
// range の境界はインフラ層が From / To の暦日から算出します。
type Period struct {
	// Kind は、集計対象期間の区分です。
	Kind PeriodKind
	// From / To は、Kind が PeriodRange のときの開始日・終了日です（両端を含みます）。
	// 暦日としての年月日のみが意味を持ち、時刻部分は解釈しません。他の区分では参照しません。
	From time.Time
	To   time.Time
}

// DashboardQueryService は、購入・商品を横断した admin ダッシュボード向け集計の参照を提供するクエリサービスです。
// 各集計は独立した投影であり、単一のスナップショットで整合することは保証しません。
type DashboardQueryService interface {
	// SummarizeSales は、指定期間に注文された購入の売上合計と件数を返します。
	// キャンセル済みの購入は除外し、未払いの購入は含みます。対象がない場合はゼロ値を返します。
	SummarizeSales(ctx context.Context, period Period) (SalesResult, error)
	// CountPurchasesByStatus は、指定期間に注文された購入のステータス別件数を購入ステータスマスタの表示順で返します。
	// SummarizeSales と異なりキャンセル済みも 1 ステータスとして含みます。
	// 期間内に購入が存在しない場合は空スライスを返します（エラーとしません）。
	CountPurchasesByStatus(ctx context.Context, period Period) ([]PurchaseStatusCountResult, error)
}

// SalesResult は、売上集計の結果です。Amount は決済スケールの整数（USD セント）です。
type SalesResult struct {
	Amount int64
	Count  int64
}

// PurchaseStatusCountResult は、1 ステータス分の購入件数の集計結果です。
// StatusID / StatusName は、購入ステータスマスタで解決済みの ID と名称です。
type PurchaseStatusCountResult struct {
	StatusID   uuid.UUID
	StatusName string
	Count      int64
}
