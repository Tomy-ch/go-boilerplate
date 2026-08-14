package summary

import (
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	clocktestkit "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/purchase/period"
	"go-boilerplate/internal/usecase/purchase/query"
	mock_query "go-boilerplate/internal/usecase/purchase/query/mock"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// summaryLoc は、注入されたロケーションが使われることを設定値と独立に固定するための、UTC から離れた固定ゾーンです。
var summaryLoc = time.FixedZone("TEST+09", 9*60*60)

// summaryNow は、summaryLoc で 2026-01-31 12:00 に相当する時刻です。UTC のままだと暦日が 1 日ずれる位置を選んでいます。
var summaryNow = time.Date(2026, time.January, 31, 3, 0, 0, 0, time.UTC)

func newAuthn(t *testing.T, userID uuid.UUID) *authbd.Authn {
	t.Helper()
	a, err := authbd.New("sub-"+userID.String(), authbd.IssuerMock, nil, nil)
	require.NoError(t, err)

	resolved, err := a.WithUserID(userID)
	require.NoError(t, err)
	return resolved
}

func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.Parse(s)
	require.NoError(t, err)
	return d
}

func Test_usecase_GetPurchaseSummary(t *testing.T) {
	t.Parallel()

	newUsecase := func(t *testing.T, qs query.PurchaseSummaryQueryService) *usecase {
		t.Helper()
		return &usecase{
			tracer: observability.NewNoopTracerFactory(t).Usecase(),
			qs:     qs,
			clk:    clocktestkit.NewMockClock(t, summaryNow),
			loc:    summaryLoc,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のuserIDをQSへ渡し集計を総計へ畳み込む", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_user")
			unprocessedID := uuidtestkit.NewTestFromSalt(t, "sm_unprocessed")
			completedID := uuidtestkit.NewTestFromSalt(t, "sm_completed")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).Return([]query.PurchaseStatusSummaryReadModel{
				{StatusID: unprocessedID, StatusName: "未処理", Count: 2, TotalAmount: 300},
				{StatusID: completedID, StatusName: "完了", Count: 1, TotalAmount: 150},
			}, nil)
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(dec(t, "4.50"), nil)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{})
			require.NoError(t, err)

			assert.Equal(t, int64(3), actual.TotalCount)
			assert.Equal(t, int64(450), actual.TotalAmount)
			assert.Equal(t, "4.5", actual.ItemsTotal.String())
			assert.Equal(t, []StatusCountView{
				{StatusID: unprocessedID, StatusName: "未処理", Count: 2, TotalAmount: 300},
				{StatusID: completedID, StatusName: "完了", Count: 1, TotalAmount: 150},
			}, actual.StatusBreakdown)
		})

		t.Run("グループ化しないとき商品単位の行を読まずgroupsを返さない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_nogroup_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).Return(nil, nil)
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(decimal.Decimal{}, nil)
			qs.EXPECT().SummarizeItemsByProductByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{})
			require.NoError(t, err)
			assert.Nil(t, actual.Groups)
		})

		t.Run("解決済みの対象期間がQSへ渡りレスポンスにも載る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_period_user")
			days := 10

			var captured period.Window
			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).DoAndReturn(
				func(_ any, _ uuid.UUID, w period.Window) ([]query.PurchaseStatusSummaryReadModel, error) {
					captured = w
					return nil, nil
				},
			)
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(decimal.Decimal{}, nil)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{
				Period: period.Spec{Kind: period.KindRecent, Days: &days},
			})
			require.NoError(t, err)

			// 相対指定は summaryNow の暦日（1/31）を終了日とし、10 日前の 1/21 が開始日になる。
			assert.True(t, captured.Filtered())
			assert.True(t, time.Date(2026, time.January, 21, 0, 0, 0, 0, summaryLoc).Equal(captured.From()))
			assert.True(t, time.Date(2026, time.January, 31, 0, 0, 0, 0, summaryLoc).Equal(captured.To()))
			// QS へ渡した期間とレスポンスの期間が同一であることを固定する（クライアントは相対指定を自前で解決しない）。
			assert.Equal(t, captured, actual.Period)
		})

		t.Run("グループ化しないとき対象期間は絞り込みなしとして返る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_allperiod_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).Return(nil, nil)
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(decimal.Decimal{}, nil)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{})
			require.NoError(t, err)
			assert.False(t, actual.Period.Filtered())
		})

		t.Run("購入が1件もない場合はnilではない空の内訳とゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_empty_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).Return([]query.PurchaseStatusSummaryReadModel{}, nil)
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(decimal.Decimal{}, nil)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{})
			require.NoError(t, err)
			assert.Equal(t, int64(0), actual.TotalCount)
			assert.Equal(t, int64(0), actual.TotalAmount)
			assert.Equal(t, "0", actual.ItemsTotal.String())
			assert.NotNil(t, actual.StatusBreakdown)
			assert.Empty(t, actual.StatusBreakdown)
		})

		t.Run("グループ化を指定したとき明細合計は商品単位の行から導かれる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_group_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).Return(nil, nil)
			// グループ化時は合計も同じ行から導くため、平坦な合計クエリは呼ばれない。
			qs.EXPECT().SumItemsByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			qs.EXPECT().SummarizeItemsByProductByUserID(gomock.Any(), userID, gomock.Any()).Return(
				[]query.PurchaseItemSummaryReadModel{
					{CategoryName: "電子機器", ProductID: uuidtestkit.NewTestFromSalt(t, "sm_p1"), ProductName: "ノートPC", ItemsTotal: dec(t, "980.00")},
					{CategoryName: "書籍", ProductID: uuidtestkit.NewTestFromSalt(t, "sm_p2"), ProductName: "技術書", ItemsTotal: dec(t, "20.50")},
				}, nil)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{
				GroupBy: []GroupKind{GroupByCategory},
			})
			require.NoError(t, err)

			assert.Equal(t, "1000.5", actual.ItemsTotal.String())
			require.Len(t, actual.Groups, 2)
			assert.Equal(t, "980", actual.Groups["電子機器"].ItemsTotal.String())
			assert.Equal(t, "20.5", actual.Groups["書籍"].ItemsTotal.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証コンテキストがnilの場合、ErrUnauthenticatedを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), nil, GetSummaryParams{})
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("内部UserIDが未解決の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)

			// WithUserID を経ていない Authn は内部 UserID を解決できない。
			authn, err := authbd.New("sub-unresolved", authbd.IssuerMock, nil, nil)
			require.NoError(t, err)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), authn, GetSummaryParams{})
			require.ErrorIs(t, err, authbd.ErrUserIDUnresolved)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("期間の必須指定が欠けているときInvalidArgumentを返しQSを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_badperiod_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{
				Period: period.Spec{Kind: period.KindRange},
			})
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("QSがエラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_err_user")
			expected := xerrors.Wrap(apperror.ErrInternal, "query service failed")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).Return(nil, expected)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{})
			require.ErrorIs(t, err, expected)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("明細合計のQSがエラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_items_err_user")
			expected := xerrors.Wrap(apperror.ErrInternal, "items query failed")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), userID, gomock.Any()).Return(nil, nil)
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(decimal.Decimal{}, expected)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(t.Context(), newAuthn(t, userID), GetSummaryParams{})
			require.ErrorIs(t, err, expected)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("グループ化に未知の単位があるときInvalidArgumentを返しQSを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_badgroup_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(
				t.Context(), newAuthn(t, userID), GetSummaryParams{GroupBy: []GroupKind{"unknown"}},
			)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Equal(t, SummaryView{}, actual)
		})

		t.Run("グループ化に同一単位の重複があるときInvalidArgumentを返しQSを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			userID := uuidtestkit.NewTestFromSalt(t, "sm_dupgroup_user")

			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			qs.EXPECT().SummarizeByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			actual, err := newUsecase(t, qs).GetPurchaseSummary(
				t.Context(), newAuthn(t, userID),
				GetSummaryParams{GroupBy: []GroupKind{GroupByCategory, GroupByCategory}},
			)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			assert.Equal(t, SummaryView{}, actual)
		})
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を注入したユースケース実装を生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			qs := mock_query.NewMockPurchaseSummaryQueryService(ctrl)
			tf := observability.NewNoopTracerFactory(t)
			clk := clocktestkit.NewMockClock(t, summaryNow)

			actual, ok := New(qs, clk, summaryLoc, tf).(*usecase)
			require.True(t, ok)
			assert.Equal(t, qs, actual.qs)
			assert.Equal(t, clk, actual.clk)
			assert.Equal(t, summaryLoc, actual.loc)
			assert.NotNil(t, actual.tracer)
		})
	})
}

func Test_toSummaryView(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステータス別の件数と金額を総計へ畳み込み内訳の順序を保持する", func(t *testing.T) {
			t.Parallel()

			first := uuidtestkit.NewTestFromSalt(t, "tv_first")
			second := uuidtestkit.NewTestFromSalt(t, "tv_second")

			actual := toSummaryView([]query.PurchaseStatusSummaryReadModel{
				{StatusID: first, StatusName: "未処理", Count: 1, TotalAmount: 100},
				{StatusID: second, StatusName: "完了", Count: 3, TotalAmount: 900},
			})

			assert.Equal(t, int64(4), actual.TotalCount)
			assert.Equal(t, int64(1000), actual.TotalAmount)
			assert.Equal(t, []StatusCountView{
				{StatusID: first, StatusName: "未処理", Count: 1, TotalAmount: 100},
				{StatusID: second, StatusName: "完了", Count: 3, TotalAmount: 900},
			}, actual.StatusBreakdown)
		})

		t.Run("集計結果が空の場合はnilではない空の内訳とゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			actual := toSummaryView(nil)

			assert.Equal(t, int64(0), actual.TotalCount)
			assert.Equal(t, int64(0), actual.TotalAmount)
			assert.NotNil(t, actual.StatusBreakdown)
			assert.Empty(t, actual.StatusBreakdown)
		})
	})
}

func Test_toGroups(t *testing.T) {
	t.Parallel()

	laptopID := uuidtestkit.NewTestFromSalt(t, "tg_laptop")
	tabletID := uuidtestkit.NewTestFromSalt(t, "tg_tablet")
	bookID := uuidtestkit.NewTestFromSalt(t, "tg_book")

	rows := func(t *testing.T) []query.PurchaseItemSummaryReadModel {
		t.Helper()
		return []query.PurchaseItemSummaryReadModel{
			{CategoryName: "電子機器", ProductID: laptopID, ProductName: "ノートPC", ItemsTotal: dec(t, "980.00")},
			{CategoryName: "電子機器", ProductID: tabletID, ProductName: "タブレット", ItemsTotal: dec(t, "300.25")},
			{CategoryName: "書籍", ProductID: bookID, ProductName: "技術書", ItemsTotal: dec(t, "20.50")},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カテゴリ単位では名称をキーに同カテゴリの商品を合算する", func(t *testing.T) {
			t.Parallel()

			total, groups := toGroups(rows(t), []GroupKind{GroupByCategory})

			assert.Equal(t, "1300.75", total.String())
			require.Len(t, groups, 2)
			assert.Equal(t, "電子機器", groups["電子機器"].Name)
			assert.Equal(t, "1280.25", groups["電子機器"].ItemsTotal.String())
			assert.Equal(t, "20.5", groups["書籍"].ItemsTotal.String())
			// 最下位の階層は下位グループを持たない。
			assert.Nil(t, groups["電子機器"].Groups)
		})

		t.Run("指定順がネストの階層順になり親は子の総和を持つ", func(t *testing.T) {
			t.Parallel()

			total, groups := toGroups(rows(t), []GroupKind{GroupByCategory, GroupByProduct})

			electronics := groups["電子機器"]
			assert.Equal(t, "1280.25", electronics.ItemsTotal.String())
			require.Len(t, electronics.Groups, 2)
			assert.Equal(t, "980", electronics.Groups[laptopID.String()].ItemsTotal.String())
			assert.Equal(t, "300.25", electronics.Groups[tabletID.String()].ItemsTotal.String())

			// どの階層でも内訳の総和が親と一致する（丸めを挟まないため誤差は生じない）。
			sum := electronics.Groups[laptopID.String()].ItemsTotal.Add(electronics.Groups[tabletID.String()].ItemsTotal)
			assert.True(t, sum.Equal(electronics.ItemsTotal))
			assert.True(t, electronics.ItemsTotal.Add(groups["書籍"].ItemsTotal).Equal(total))
		})

		t.Run("対象行が無いときnilではない空のマップとゼロ値を返す", func(t *testing.T) {
			t.Parallel()

			total, groups := toGroups(nil, []GroupKind{GroupByCategory})

			assert.Equal(t, "0", total.String())
			assert.NotNil(t, groups)
			assert.Empty(t, groups)
		})
	})
}

func Test_levelsOf(t *testing.T) {
	t.Parallel()

	productID := uuidtestkit.NewTestFromSalt(t, "lv_product")
	row := query.PurchaseItemSummaryReadModel{
		CategoryName: "電子機器",
		ProductID:    productID,
		ProductName:  "ノートPC",
		ItemsTotal:   decimal.Decimal{},
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("カテゴリは名称をそのままキーにする", func(t *testing.T) {
			t.Parallel()

			// product_categories.name は UNIQUE なので、名称のままでもグループが衝突しない。
			assert.Equal(t, []groupLevel{{key: "電子機器", name: "電子機器"}}, levelsOf(row, []GroupKind{GroupByCategory}))
		})

		t.Run("商品はIDをキーにし表示名を別に持つ", func(t *testing.T) {
			t.Parallel()

			// products.name に一意制約は無く、名称をキーにすると同名の別商品が 1 グループへ畳み込まれてしまう。
			assert.Equal(t,
				[]groupLevel{{key: productID.String(), name: "ノートPC"}},
				levelsOf(row, []GroupKind{GroupByProduct}),
			)
		})

		t.Run("指定順がそのまま階層の順序になる", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				[]groupLevel{{key: productID.String(), name: "ノートPC"}, {key: "電子機器", name: "電子機器"}},
				levelsOf(row, []GroupKind{GroupByProduct, GroupByCategory}),
			)
		})

		t.Run("グループ化単位が空のとき階層を持たない", func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, levelsOf(row, nil))
		})

		t.Run("未知の単位は階層に含めず黙って読み飛ばす", func(t *testing.T) {
			t.Parallel()

			// 本来 validateGroupBy が先に弾くため到達しない。その防御が外れたときに
			// 「その階層が消えて内訳の総和が親と食い違う」形で壊れることを固定しておく。
			assert.Empty(t, levelsOf(row, []GroupKind{"unknown"}))
			assert.Equal(t,
				[]groupLevel{{key: "電子機器", name: "電子機器"}},
				levelsOf(row, []GroupKind{"unknown", GroupByCategory}),
			)
		})
	})
}

func Test_accumulate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("階層が空のときマップをそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, accumulate(nil, nil, dec(t, "10.00")))
		})

		t.Run("nilのマップからでもノードを作る", func(t *testing.T) {
			t.Parallel()

			groups := accumulate(nil, []groupLevel{{key: "k", name: "表示名"}}, dec(t, "10.00"))

			require.Len(t, groups, 1)
			assert.Equal(t, "表示名", groups["k"].Name)
			assert.Equal(t, "10", groups["k"].ItemsTotal.String())
		})

		t.Run("同じキーの行は既存ノードへ加算される", func(t *testing.T) {
			t.Parallel()

			groups := accumulate(nil, []groupLevel{{key: "k", name: "表示名"}}, dec(t, "10.00"))
			groups = accumulate(groups, []groupLevel{{key: "k", name: "表示名"}}, dec(t, "0.50"))

			require.Len(t, groups, 1)
			assert.Equal(t, "10.5", groups["k"].ItemsTotal.String())
		})

		t.Run("下位階層へ加算しても上位ノードが総和を持つ", func(t *testing.T) {
			t.Parallel()

			levels := func(child string) []groupLevel {
				return []groupLevel{{key: "parent", name: "親"}, {key: child, name: child}}
			}
			groups := accumulate(nil, levels("a"), dec(t, "10.00"))
			groups = accumulate(groups, levels("b"), dec(t, "5.00"))

			parent := groups["parent"]
			assert.Equal(t, "15", parent.ItemsTotal.String())
			require.Len(t, parent.Groups, 2)
			assert.Equal(t, "10", parent.Groups["a"].ItemsTotal.String())
			assert.Equal(t, "5", parent.Groups["b"].ItemsTotal.String())
		})
	})
}

func Test_validateGroupBy(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定を受理する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateGroupBy(nil))
		})

		t.Run("カテゴリのみの指定を受理する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateGroupBy([]GroupKind{GroupByCategory}))
		})

		t.Run("商品のみの指定を受理する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateGroupBy([]GroupKind{GroupByProduct}))
		})

		t.Run("カテゴリと商品を重複なく並べた指定を受理する", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateGroupBy([]GroupKind{GroupByCategory, GroupByProduct}))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未知の単位はInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			require.ErrorIs(t, validateGroupBy([]GroupKind{"unknown"}), apperror.ErrInvalidArgument)
		})

		t.Run("同一単位の重複はInvalidArgumentを返す", func(t *testing.T) {
			t.Parallel()

			// 重複を通すと同じ階層が入れ子になり、内訳の総和が親と一致しなくなる。
			require.ErrorIs(t,
				validateGroupBy([]GroupKind{GroupByCategory, GroupByCategory}),
				apperror.ErrInvalidArgument,
			)
		})
	})
}

func Test_usecase_summarizeItems(t *testing.T) {
	t.Parallel()

	userID := uuidtestkit.NewTestFromSalt(t, "si_user")

	newUsecase := func(t *testing.T, qs query.PurchaseSummaryQueryService) *usecase {
		t.Helper()
		return &usecase{tracer: observability.NewNoopTracerFactory(t).Usecase(), qs: qs}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("グループ化しないとき平坦な合計だけを取り内訳を返さない", func(t *testing.T) {
			t.Parallel()

			qs := mock_query.NewMockPurchaseSummaryQueryService(gomock.NewController(t))
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(dec(t, "12.50"), nil)
			qs.EXPECT().SummarizeItemsByProductByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			total, groups, err := newUsecase(t, qs).summarizeItems(t.Context(), userID, period.Window{}, nil)
			require.NoError(t, err)
			assert.Equal(t, "12.5", total.String())
			assert.Nil(t, groups)
		})

		t.Run("グループ化するとき商品単位の行から合計と内訳を導く", func(t *testing.T) {
			t.Parallel()

			qs := mock_query.NewMockPurchaseSummaryQueryService(gomock.NewController(t))
			qs.EXPECT().SumItemsByUserID(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			qs.EXPECT().SummarizeItemsByProductByUserID(gomock.Any(), userID, gomock.Any()).Return(
				[]query.PurchaseItemSummaryReadModel{
					{CategoryName: "書籍", ProductID: uuidtestkit.NewTestFromSalt(t, "si_book"), ProductName: "技術書", ItemsTotal: dec(t, "20.50")},
				}, nil)

			total, groups, err := newUsecase(t, qs).summarizeItems(
				t.Context(), userID, period.Window{}, []GroupKind{GroupByCategory})
			require.NoError(t, err)
			assert.Equal(t, "20.5", total.String())
			require.Len(t, groups, 1)
			assert.Equal(t, "20.5", groups["書籍"].ItemsTotal.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("平坦な合計のQSのエラーが伝播する", func(t *testing.T) {
			t.Parallel()

			qs := mock_query.NewMockPurchaseSummaryQueryService(gomock.NewController(t))
			qs.EXPECT().SumItemsByUserID(gomock.Any(), userID, gomock.Any()).Return(decimal.Decimal{}, apperror.ErrInternal)

			_, groups, err := newUsecase(t, qs).summarizeItems(t.Context(), userID, period.Window{}, nil)
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Nil(t, groups)
		})

		t.Run("商品単位の集計のQSのエラーが伝播する", func(t *testing.T) {
			t.Parallel()

			qs := mock_query.NewMockPurchaseSummaryQueryService(gomock.NewController(t))
			qs.EXPECT().SummarizeItemsByProductByUserID(gomock.Any(), userID, gomock.Any()).Return(nil, apperror.ErrInternal)

			_, groups, err := newUsecase(t, qs).summarizeItems(
				t.Context(), userID, period.Window{}, []GroupKind{GroupByCategory})
			require.ErrorIs(t, err, apperror.ErrInternal)
			assert.Nil(t, groups)
		})
	})
}
