//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package query は、admin ダッシュボードの横断集計のクエリサービスインターフェースを提供します。
package query

import (
	"context"
	"time"

	"go-boilerplate/pkg/uuid"
)

// Window は、解決済みの集計対象期間です。半開区間 [After, Before) で表し、Before は終了日の翌日です。
// 相対指定や暦月指定から暦日への解決は usecase 層が済ませているため、クエリサービスは境界をそのまま
// SQL の述語へ渡します。
type Window struct {
	// After は、集計対象に含める最初の時刻です（この時刻を含みます）。
	After time.Time
	// Before は、集計対象に含めない最初の時刻です（この時刻を含みません）。
	Before time.Time
}

// DashboardQueryService は、購入・商品を横断した admin ダッシュボード向け集計の参照を提供するクエリサービスです。
// 各集計は独立した投影であり、単一のスナップショットで整合することは保証しません。
type DashboardQueryService interface {
	// SummarizeSales は、指定期間に注文された購入の売上合計と件数を返します。
	// キャンセル済みの購入は除外し、未払いの購入は含みます。対象がない場合はゼロ値を返します。
	SummarizeSales(ctx context.Context, window Window) (SalesResult, error)
	// CountPurchasesByStatus は、指定期間に注文された購入のステータス別件数を購入ステータスマスタの表示順で返します。
	// SummarizeSales と異なりキャンセル済みも 1 ステータスとして含みます。
	// 期間内に購入が存在しない場合は空スライスを返します（エラーとしません）。
	CountPurchasesByStatus(ctx context.Context, window Window) ([]PurchaseStatusCountResult, error)
}

// SalesResult は、売上集計の結果です。Amount は決済スケールの整数（USD セント）です。
type SalesResult struct {
	Amount int64
	Count  int64
}

// PurchaseStatusCountResult は、1 ステータス分の購入件数の集計結果です。
// StatusID / StatusName は、購入ステータスマスタで解決済みの ID と名称です。
// StatusCode は、購入ステータスの業務キー（Status.Code）です。
type PurchaseStatusCountResult struct {
	StatusID   uuid.UUID
	StatusCode int
	StatusName string
	Count      int64
}
