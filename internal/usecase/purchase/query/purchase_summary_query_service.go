//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

package query

import (
	"context"

	"go-boilerplate/pkg/uuid"
)

// PurchaseSummaryQueryService は、購入の集計 read 投影を提供する QueryService です。
// 件数・合計金額・ステータス別内訳は購入集約を再構成できない派生投影であるため、Repository ではなく
// 読み取り側に置きます（ADR-0027）。
type PurchaseSummaryQueryService interface {
	// SummarizeByUserID は、認証主体（userID）の購入をステータス単位に集計し、購入ステータスマスタの表示順で返します。
	// 所有権は SQL の WHERE 述語で担保します。対象の購入が存在しない場合は空スライスを返します（エラーとしません）。
	// キャンセル済みの購入も集計対象に含みます。
	SummarizeByUserID(ctx context.Context, userID uuid.UUID) ([]PurchaseStatusSummaryReadModel, error)
}

// PurchaseStatusSummaryReadModel は、1 ステータス分の購入集計の読み取りモデルです。
// 金額は決済スケールの整数（USD セント）です。
type PurchaseStatusSummaryReadModel struct {
	// StatusID / StatusName は、購入ステータスマスタで解決済みの ID と名称です。
	StatusID   uuid.UUID
	StatusName string
	// Count は、当該ステータスの購入件数です。
	Count int64
	// TotalAmount は、当該ステータスの購入金額の合計です。
	TotalAmount int64
}
