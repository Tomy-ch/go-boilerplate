package usersmepurchases

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/users/me/purchases/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/purchase/period"
	summaryuc "go-boilerplate/internal/usecase/purchase/summary"
	mock_summaryuc "go-boilerplate/internal/usecase/purchase/summary/mock"
	"go-boilerplate/pkg/decimal"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// testLoc は、注入されたロケーションが使われることを設定値と独立に固定するための、UTC から離れた固定ゾーンです。
var testLoc = time.FixedZone("TEST+09", 9*60*60)

// authnContext は、内部ユーザー ID を解決済みの認証コンテキストを返すテストヘルパーです。
func authnContext(t *testing.T, userID uuid.UUID) context.Context {
	t.Helper()
	ctx := ctxhelper.WithAuthn(context.Background())
	authn, err := auth.New("subject", "issuer", nil, nil)
	require.NoError(t, err)

	resolved, err := authn.WithUserID(userID)
	require.NoError(t, err)
	require.True(t, ctxhelper.SetAuthn(ctx, *resolved))
	return ctx
}

func dec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.Parse(s)
	require.NoError(t, err)
	return d
}

// testWindow は、両端を含む暦日から絞り込み済みの対象期間を組み立てるテストヘルパーです。
func testWindow(t *testing.T, from, to time.Time) period.Window {
	t.Helper()
	w, err := period.Resolve(period.Spec{Kind: period.KindRange, From: &from, To: &to}, time.Time{}, testLoc)
	require.NoError(t, err)
	return w
}

// summaryViewFixture は、購入集計ビューを生成するテストヘルパーです。
func summaryViewFixture(t *testing.T) summaryuc.SummaryView {
	t.Helper()
	return summaryuc.SummaryView{
		Period: testWindow(t,
			time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC),
			time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC),
		),
		TotalCount:  3,
		TotalAmount: 450,
		ItemsTotal:  dec(t, "4.50"),
		StatusBreakdown: []summaryuc.StatusCountView{
			{
				StatusID:    uuidtestkit.NewTestFromSalt(t, "hs_unprocessed"),
				StatusCode:  1,
				StatusName:  "未処理",
				Count:       2,
				TotalAmount: 300,
			},
			{
				StatusID:    uuidtestkit.NewTestFromSalt(t, "hs_paid"),
				StatusCode:  7,
				StatusName:  "支払い済み",
				Count:       1,
				TotalAmount: 150,
			},
		},
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	uc := mock_summaryuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, uc)

	routes := e.Router().Routes()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodGet, routes[0].Method)
	assert.Equal(t, "/v1/users/me/purchases/summary", routes[0].Path)
}

func Test_server_GetUsersMePurchasesSummary(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証主体のAuthnをユースケースへ渡し集計を200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "hs_user")
			view := summaryViewFixture(t)
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, authn *auth.Authn, _ summaryuc.GetSummaryParams) (summaryuc.SummaryView, error) {
					uid, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, uid)
					return view, nil
				})

			resp, err := s.GetUsersMePurchasesSummary(authnContext(t, userID), gen.GetUsersMePurchasesSummaryRequestObject{})
			require.NoError(t, err)

			r, ok := resp.(gen.GetUsersMePurchasesSummary200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, int64(3), r.TotalCount)
			assert.Equal(t, int64(450), r.TotalAmount)
			assert.Equal(t, "4.5", r.ItemsTotal)
			require.Len(t, r.StatusBreakdown, 2)
			assert.Equal(t, "未処理", r.StatusBreakdown[0].Status.Name)
			assert.Equal(t, int64(2), r.StatusBreakdown[0].Count)
			assert.Equal(t, int64(300), r.StatusBreakdown[0].TotalAmount)
		})

		t.Run("期間とグループ化のパラメータがユースケースへ渡る", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			kind := gen.GetUsersMePurchasesSummaryParamsPeriod(period.KindRecent)
			days := int32(10)
			groupBy := gen.PurchaseGroupByParam{"category", "product"}

			var captured summaryuc.GetSummaryParams
			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ *auth.Authn, params summaryuc.GetSummaryParams) (summaryuc.SummaryView, error) {
					captured = params
					return summaryuc.SummaryView{}, nil
				})

			_, err := s.GetUsersMePurchasesSummary(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "hs_params_user")),
				gen.GetUsersMePurchasesSummaryRequestObject{
					Params: gen.GetUsersMePurchasesSummaryParams{Period: &kind, Days: &days, GroupBy: &groupBy},
				},
			)
			require.NoError(t, err)

			assert.Equal(t, period.KindRecent, captured.Period.Kind)
			require.NotNil(t, captured.Period.Days)
			assert.Equal(t, 10, *captured.Period.Days)
			// 指定順がネストの階層順になるため、並びが保たれることを固定する。
			assert.Equal(t, []summaryuc.GroupKind{summaryuc.GroupByCategory, summaryuc.GroupByProduct}, captured.GroupBy)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が無い場合、ErrUnauthenticatedUserを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.GetUsersMePurchasesSummary(context.Background(), gen.GetUsersMePurchasesSummaryRequestObject{})
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_summaryuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().GetPurchaseSummary(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(summaryuc.SummaryView{}, apperror.ErrInternal)

			_, err := s.GetUsersMePurchasesSummary(authnContext(t, uuidtestkit.NewTestFromSalt(t, "hs_user_err")),
				gen.GetUsersMePurchasesSummaryRequestObject{})
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_toPurchaseAggregateResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("集計ビューをステータス別内訳込みのレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := summaryViewFixture(t)
			r := toPurchaseAggregateResponse(view)

			assert.Equal(t, view.TotalCount, r.TotalCount)
			assert.Equal(t, view.TotalAmount, r.TotalAmount)
			assert.Equal(t, view.ItemsTotal.String(), r.ItemsTotal)
			// 対象期間の写像そのものは Test_toPurchasePeriodResponse が持つため、ここでは配線だけを固定する。
			assert.NotNil(t, r.Period.From)
			require.Len(t, r.StatusBreakdown, len(view.StatusBreakdown))
			for i, b := range view.StatusBreakdown {
				assert.Equal(t, b.StatusID.ToPrimitive(), r.StatusBreakdown[i].Status.Id)
				assert.Equal(t, b.StatusName, r.StatusBreakdown[i].Status.Name)
				assert.EqualValues(t, b.StatusCode, r.StatusBreakdown[i].Status.Code)
				assert.Equal(t, b.Count, r.StatusBreakdown[i].Count)
				assert.Equal(t, b.TotalAmount, r.StatusBreakdown[i].TotalAmount)
			}
		})

		t.Run("内訳が空の場合はnilではない空配列のレスポンスを返す", func(t *testing.T) {
			t.Parallel()

			r := toPurchaseAggregateResponse(summaryuc.SummaryView{})

			assert.Equal(t, int64(0), r.TotalCount)
			assert.Equal(t, int64(0), r.TotalAmount)
			assert.Equal(t, "0", r.ItemsTotal)
			assert.NotNil(t, r.StatusBreakdown)
			assert.Empty(t, r.StatusBreakdown)
		})
	})
}

func Test_toPurchasePeriodResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解決済みの対象期間を両端の暦日で返す", func(t *testing.T) {
			t.Parallel()

			r := toPurchasePeriodResponse(testWindow(t,
				time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC),
				time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC),
			))

			require.NotNil(t, r.From)
			require.NotNil(t, r.To)
			assert.Equal(t, "2026-01-21", r.From.Format(time.DateOnly))
			assert.Equal(t, "2026-01-31", r.To.Format(time.DateOnly))
		})

		t.Run("全期間を集計した場合は両端をnullで返す", func(t *testing.T) {
			t.Parallel()

			r := toPurchasePeriodResponse(period.Window{})

			assert.Nil(t, r.From)
			assert.Nil(t, r.To)
		})
	})
}

func Test_toGroupsResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("グループ化していない場合はnilを返しレスポンスにgroupsを含めない", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, toGroupsResponse(nil))
		})

		t.Run("グループ化を指定して対象が無い場合は空オブジェクトを返す", func(t *testing.T) {
			t.Parallel()

			r := toGroupsResponse(map[string]summaryuc.GroupNodeView{})

			require.NotNil(t, r)
			assert.Empty(t, *r)
		})

		t.Run("入れ子の集計を階層を保ったまま写像する", func(t *testing.T) {
			t.Parallel()

			productID := uuidtestkit.NewTestFromSalt(t, "hs_product").String()
			r := toGroupsResponse(map[string]summaryuc.GroupNodeView{
				"電子機器": {
					Name:       "電子機器",
					ItemsTotal: dec(t, "980.00"),
					Groups: map[string]summaryuc.GroupNodeView{
						productID: {Name: "ノートPC", ItemsTotal: dec(t, "980.00")},
					},
				},
			})

			require.NotNil(t, r)
			groups := *r
			require.Contains(t, groups, "電子機器")
			assert.Equal(t, "電子機器", groups["電子機器"].Name)
			assert.Equal(t, "980", groups["電子機器"].ItemsTotal)

			require.NotNil(t, groups["電子機器"].Groups)
			children := *groups["電子機器"].Groups
			require.Contains(t, children, productID)
			assert.Equal(t, "ノートPC", children[productID].Name)
		})
	})
}

func Test_toSubGroupsResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("下位単位の指定が無い場合はnilを返しgroupsを含めない", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, toSubGroupsResponse(nil))
		})

		t.Run("下位ノードを表示名と金額へ写像する", func(t *testing.T) {
			t.Parallel()

			productID := uuidtestkit.NewTestFromSalt(t, "hs_subgroup_product").String()
			r := toSubGroupsResponse(map[string]summaryuc.GroupNodeView{
				productID: {Name: "ノートPC", ItemsTotal: dec(t, "980.00")},
			})

			require.NotNil(t, r)
			require.Contains(t, *r, productID)
			assert.Equal(t, "ノートPC", (*r)[productID].Name)
			assert.Equal(t, "980", (*r)[productID].ItemsTotal)
		})
	})
}

func Test_toPeriodSpec(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期間パラメータが漏れなくユースケースの期間指定へ写像される", func(t *testing.T) {
			t.Parallel()

			kind := gen.GetUsersMePurchasesSummaryParamsPeriod(period.KindRange)
			from := openapi_types.Date{Time: time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC)}
			to := openapi_types.Date{Time: time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)}
			month := "2026-01"
			days := int32(10)

			actual := toPeriodSpec(gen.GetUsersMePurchasesSummaryParams{
				Period: &kind,
				From:   &from,
				To:     &to,
				Month:  &month,
				Days:   &days,
			})

			assert.Equal(t, period.KindRange, actual.Kind)
			require.NotNil(t, actual.From)
			require.NotNil(t, actual.To)
			assert.True(t, from.Equal(*actual.From))
			assert.True(t, to.Equal(*actual.To))
			assert.Equal(t, &month, actual.Month)
			require.NotNil(t, actual.Days)
			assert.Equal(t, 10, *actual.Days)
		})

		t.Run("パラメータ未指定のときゼロ値の期間指定になる", func(t *testing.T) {
			t.Parallel()

			// ゼロ値は全期間を意味するため、既定の呼び出しが期間で絞り込まれないことを固定する。
			assert.Equal(t, period.Spec{}, toPeriodSpec(gen.GetUsersMePurchasesSummaryParams{}))
		})
	})
}

func Test_toGroupKinds(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未指定のときnilを返しグループ化しないことを表す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, toGroupKinds(nil))
		})

		t.Run("指定順を保ったままユースケースの指定へ写像する", func(t *testing.T) {
			t.Parallel()

			groupBy := gen.PurchaseGroupByParam{"product", "category"}
			assert.Equal(t,
				[]summaryuc.GroupKind{summaryuc.GroupByProduct, summaryuc.GroupByCategory},
				toGroupKinds(&groupBy),
			)
		})
	})
}
