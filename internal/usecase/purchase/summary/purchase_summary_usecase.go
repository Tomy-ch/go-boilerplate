//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package summary は、認証主体自身の購入集計の参照ユースケースを提供します。
package summary

import (
	"context"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// StatusCountView は、ステータス別内訳 1 件分のユースケース出力 DTO です。
// ステータスは購入ステータスマスタで解決済みの ID と名称、TotalAmount は USD セント単位の整数です。
type StatusCountView struct {
	StatusID    uuid.UUID
	StatusName  string
	Count       int64
	TotalAmount int64
}

// SummaryView は、購入集計のユースケース出力 DTO です。TotalAmount は USD セント単位の整数で、
// キャンセル済みの購入も総件数・合計金額に含みます（キャンセルは StatusBreakdown の 1 要素として返ります）。
type SummaryView struct {
	// TotalCount / TotalAmount は、購入総件数と購入金額の合計です。購入がない場合はいずれも 0 です。
	TotalCount  int64
	TotalAmount int64
	// StatusBreakdown は、購入に出現したステータスのみの内訳です。購入がない場合は空スライスです。
	StatusBreakdown []StatusCountView
}

// Usecase は、認証主体自身の購入集計の参照ユースケースを定義します。
type Usecase interface {
	// GetPurchaseSummary は、認証主体自身の購入の総件数・合計金額・ステータス別内訳を返します。
	// 集計は認証主体の userID に限定され、他ユーザーの購入は含みません。購入がない場合はゼロ値を返します。
	GetPurchaseSummary(ctx context.Context, authn *auth.Authn) (SummaryView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	qs     query.PurchaseSummaryQueryService
}

// New は、購入集計の参照ユースケースを生成します。
func New(qs query.PurchaseSummaryQueryService, tf observability.TracerFactory) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		qs:     qs,
	}
}

func (u *usecase) GetPurchaseSummary(ctx context.Context, authn *auth.Authn) (SummaryView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return SummaryView{}, xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")
	}
	userID, err := authn.UserID()
	if err != nil {
		return SummaryView{}, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	results, err := u.qs.SummarizeByUserID(ctx, userID)
	if err != nil {
		return SummaryView{}, err
	}

	return toSummaryView(results), nil
}

// toSummaryView は、ステータス別の集計結果を総計へ畳み込みつつ出力 DTO へ写像します。
// 総計をステータス別集計から導くことで、総件数・合計金額と内訳が同一スナップショットで整合します。
func toSummaryView(results []query.PurchaseStatusSummaryReadModel) SummaryView {
	view := SummaryView{StatusBreakdown: make([]StatusCountView, len(results))}
	for i, r := range results {
		view.TotalCount += r.Count
		view.TotalAmount += r.TotalAmount
		view.StatusBreakdown[i] = StatusCountView{
			StatusID:    r.StatusID,
			StatusName:  r.StatusName,
			Count:       r.Count,
			TotalAmount: r.TotalAmount,
		}
	}
	return view
}
