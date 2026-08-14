//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package summary は、認証主体自身の購入集計の参照ユースケースを提供します。
package summary

import (
	"context"
	"fmt"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/boundary/clock"
	"go-boilerplate/internal/usecase/purchase/period"
	"go-boilerplate/internal/usecase/purchase/query"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

const (
	// GroupByCategory は、商品カテゴリ単位でグループ化する指定です。
	GroupByCategory GroupKind = "category"
	// GroupByProduct は、商品単位でグループ化する指定です。
	GroupByProduct GroupKind = "product"
)

// GroupKind は、集計のグループ化単位です。
type GroupKind string

// groupLevel は、集計行を 1 つのグループ化単位で見たときのマップキーと表示名です。
type groupLevel struct {
	key  string
	name string
}

// GetSummaryParams は、購入集計取得ユースケースの入力パラメータです。
type GetSummaryParams struct {
	// Period は、集計対象期間の指定です。ゼロ値は全期間を意味します。
	Period period.Spec
	// GroupBy は、グループ化単位を適用順（先頭が最上位の階層）に並べたものです。空ならグループ化しません。
	GroupBy []GroupKind
}

// StatusCountView は、ステータス別内訳 1 件分のユースケース出力 DTO です。
// ステータスは購入ステータスマスタで解決済みの ID と名称、TotalAmount は USD セント単位の整数です。
type StatusCountView struct {
	StatusID    uuid.UUID
	StatusName  string
	Count       int64
	TotalAmount int64
}

// GroupNodeView は、グループ化した集計 1 ノード分のユースケース出力 DTO です。
// ItemsTotal は価格スケールの正確な decimal（USD ドル）で、下位ノードを持つ場合はその総和と一致します。
type GroupNodeView struct {
	// Name は、グループの表示名（カテゴリ名称・商品名称）です。
	Name string
	// ItemsTotal は、当該グループの明細金額の合計です。
	ItemsTotal decimal.Decimal
	// Groups は、下位のグループ化単位ごとのノードです。最下位のノードでは nil です。
	Groups map[string]GroupNodeView
}

// SummaryView は、購入集計のユースケース出力 DTO です。集計値はすべて同一の母集団
// （認証主体の購入・キャンセル済みを除く・Period の期間内）から算出します。
type SummaryView struct {
	// Period は、集計に実際に用いた対象期間です。全期間を集計した場合は絞り込みなしを表します。
	Period period.Window
	// TotalCount / TotalAmount は、購入件数と支払金額（小計 + 税額 + 送料）の合計です。
	// 対象がない場合はいずれも 0 で、TotalAmount は USD セント単位の整数です。
	TotalCount  int64
	TotalAmount int64
	// ItemsTotal は、明細金額（単価 × 数量）の合計です。税額・送料を含まないため TotalAmount とは
	// 一致せず、価格スケールの正確な decimal（USD ドル）です。
	ItemsTotal decimal.Decimal
	// StatusBreakdown は、購入に出現したステータスのみの内訳です。対象がない場合は空スライスです。
	StatusBreakdown []StatusCountView
	// Groups は、GetSummaryParams.GroupBy の最上位単位ごとのノードです。グループ化しない場合は nil です。
	Groups map[string]GroupNodeView
}

// Usecase は、認証主体自身の購入集計の参照ユースケースを定義します。
type Usecase interface {
	// GetPurchaseSummary は、認証主体自身の購入の件数・支払金額・明細金額・ステータス別内訳を返します。
	// 集計は認証主体の userID に限定され、他ユーザーの購入は含みません。キャンセル済みの購入は除外します。
	// params.GroupBy を指定した場合は、明細金額をその単位へ分解した内訳も返します。
	// 対象の購入がない場合はゼロ値を返します。期間指定・グループ化指定が不正な場合は InvalidArgument を返します。
	GetPurchaseSummary(ctx context.Context, authn *auth.Authn, params GetSummaryParams) (SummaryView, error)
}

// usecase は、Usecase の実装です。
type usecase struct {
	tracer observability.LayerTracer
	qs     query.PurchaseSummaryQueryService
	clk    clock.Clock
	loc    *time.Location
}

// New は、購入集計の参照ユースケースを生成します。
func New(
	qs query.PurchaseSummaryQueryService,
	clk clock.Clock,
	loc *time.Location,
	tf observability.TracerFactory,
) Usecase {
	return &usecase{
		tracer: tf.Usecase(),
		qs:     qs,
		clk:    clk,
		loc:    loc,
	}
}

func (u *usecase) GetPurchaseSummary(
	ctx context.Context, authn *auth.Authn, params GetSummaryParams,
) (SummaryView, error) {
	ctx, endSpan := u.tracer.Start(ctx)
	defer endSpan()

	if authn == nil {
		return SummaryView{}, xerrors.Wrap(apperror.ErrUnauthenticated, "requires authenticated user")
	}
	userID, err := authn.UserID()
	if err != nil {
		return SummaryView{}, xerrors.Wrap(err, "failed to get user ID from authenticator")
	}

	// 対象期間はレスポンスにも含めるため、現在時刻とタイムゾーンへの依存をここで解決して確定させる。
	window, err := period.Resolve(params.Period, u.clk.Now(), u.loc)
	if err != nil {
		return SummaryView{}, err
	}
	if err = validateGroupBy(params.GroupBy); err != nil {
		return SummaryView{}, err
	}

	statuses, err := u.qs.SummarizeByUserID(ctx, userID, window)
	if err != nil {
		return SummaryView{}, err
	}

	itemsTotal, groups, err := u.summarizeItems(ctx, userID, window, params.GroupBy)
	if err != nil {
		return SummaryView{}, err
	}

	view := toSummaryView(statuses)
	view.Period = window
	view.ItemsTotal = itemsTotal
	view.Groups = groups
	return view, nil
}

// summarizeItems は、明細金額の合計と、要求されたグループ化単位の内訳を返します。
// グループ化しない呼び出しは商品単位の行を読まずに合計だけを取り、要求された場合のみ内訳を畳み込みます。
func (u *usecase) summarizeItems(
	ctx context.Context, userID uuid.UUID, window period.Window, groupBy []GroupKind,
) (decimal.Decimal, map[string]GroupNodeView, error) {
	if len(groupBy) == 0 {
		total, err := u.qs.SumItemsByUserID(ctx, userID, window)
		if err != nil {
			return decimal.Decimal{}, nil, err
		}
		return total, nil, nil
	}

	rows, err := u.qs.SummarizeItemsByProductByUserID(ctx, userID, window)
	if err != nil {
		return decimal.Decimal{}, nil, err
	}
	total, groups := toGroups(rows, groupBy)
	return total, groups, nil
}

// validateGroupBy は、グループ化単位の指定を検証します。未知の単位と同一単位の重複を拒否します。
// 重複を通すと同じ階層が入れ子になり、内訳の総和が親と一致しなくなります。
func validateGroupBy(groupBy []GroupKind) error {
	seen := make(map[GroupKind]struct{}, len(groupBy))
	for _, g := range groupBy {
		if g != GroupByCategory && g != GroupByProduct {
			return xerrors.Wrap(apperror.ErrInvalidArgument, fmt.Sprintf("unknown groupBy: %s", g))
		}
		if _, dup := seen[g]; dup {
			return xerrors.Wrap(apperror.ErrInvalidArgument, fmt.Sprintf("groupBy %s appears more than once", g))
		}
		seen[g] = struct{}{}
	}
	return nil
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

// toGroups は、商品単位の集計行を groupBy の指定順に入れ子のマップへ畳み込み、その総和も返します。
// 総和を同じ行から導くことで、明細金額の合計と内訳が必ず一致します。
func toGroups(
	rows []query.PurchaseItemSummaryReadModel, groupBy []GroupKind,
) (decimal.Decimal, map[string]GroupNodeView) {
	groups := make(map[string]GroupNodeView, len(rows))
	total := decimal.Decimal{}
	for _, row := range rows {
		groups = accumulate(groups, levelsOf(row, groupBy), row.ItemsTotal)
		total = total.Add(row.ItemsTotal)
	}
	return total, groups
}

// levelsOf は、集計行を groupBy の指定順にたどるためのキーと表示名の並びへ写像します。
// カテゴリ名は商品カテゴリマスタで一意なためそのままキーにでき、商品名は一意でないため ID をキーにします
// （同名の別商品を 1 つのグループへ畳み込まないため）。
func levelsOf(row query.PurchaseItemSummaryReadModel, groupBy []GroupKind) []groupLevel {
	levels := make([]groupLevel, 0, len(groupBy))
	for _, g := range groupBy {
		switch g {
		case GroupByCategory:
			levels = append(levels, groupLevel{key: row.CategoryName, name: row.CategoryName})
		case GroupByProduct:
			levels = append(levels, groupLevel{key: row.ProductID.String(), name: row.ProductName})
		}
	}
	return levels
}

// accumulate は、1 行の金額を levels がたどる各階層のノードへ加算し、更新後のマップを返します。
// どの階層のノードもその配下の総和を持つため、内訳の合計は常に親と一致します。
func accumulate(groups map[string]GroupNodeView, levels []groupLevel, amount decimal.Decimal) map[string]GroupNodeView {
	if len(levels) == 0 {
		return groups
	}
	if groups == nil {
		groups = make(map[string]GroupNodeView, 1)
	}
	level := levels[0]
	node := groups[level.key]
	node.Name = level.name
	node.ItemsTotal = node.ItemsTotal.Add(amount)
	node.Groups = accumulate(node.Groups, levels[1:], amount)
	groups[level.key] = node
	return groups
}
