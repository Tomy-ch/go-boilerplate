package purchases

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/idempotency"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	"go-boilerplate/internal/usecase/purchase/period"
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestSummaryView(t *testing.T) purchaseuc.PurchaseSummaryView {
	t.Helper()
	return purchaseuc.PurchaseSummaryView{
		Code:          "h-code",
		TotalAmount:   176500,
		StatusID:      uuidtestkit.NewTestFromSalt(t, "h_status"),
		StatusCode:    5,
		StatusName:    "完了",
		FirstItemName: "ワイヤレスイヤホン",
		ItemCount:     3,
		OrderedAt:     time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
	}
}

func Test_server_GetPurchases(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みユーザーの一覧をNextCursor付きHasNext=trueで返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			userID := uuidtestkit.NewTestFromSalt(t, "get_user")
			view := newTestSummaryView(t)
			nextCursor := "next-opaque-cursor"
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, uid uuid.UUID, _ *paging.Cursor, _ period.Spec) (*purchaseuc.PurchaseListView, error) {
					assert.Equal(t, userID, uid)
					return &purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{view}, NextCursor: &nextCursor}, nil
				})

			resp, err := s.GetPurchases(authnContext(t, userID), gen.GetPurchasesRequestObject{Params: gen.GetPurchasesParams{}})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPurchases200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.PurchaseListResponse{
				Items: []gen.PurchaseSummaryResponse{{
					Code:        "h-code",
					TotalAmount: 176500,
					Status: gen.PurchaseStatusRef{
						Id:   view.StatusID.ToPrimitive(),
						Code: 5,
						Name: "完了",
					},
					FirstItemName: view.FirstItemName,
					ItemCount:     int64(view.ItemCount),
					OrderedAt:     view.OrderedAt,
				}},
				NextCursor: &nextCursor,
				HasNext:    true,
			}, gen.PurchaseListResponse(actual))
		})

		t.Run("末尾ページはNextCursorがnilでHasNextがfalseになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			userID := uuidtestkit.NewTestFromSalt(t, "get_user")
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&purchaseuc.PurchaseListView{Items: []purchaseuc.PurchaseSummaryView{}, NextCursor: nil}, nil)

			resp, err := s.GetPurchases(authnContext(t, userID), gen.GetPurchasesRequestObject{Params: gen.GetPurchasesParams{}})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPurchases200JSONResponse)
			require.True(t, ok)
			assert.Empty(t, actual.Items)
			assert.Nil(t, actual.NextCursor)
			assert.False(t, actual.HasNext)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証のときErrUnauthenticatedを返しUsecaseを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			// 認証情報を仕込まない context のため GetAuthn が false を返す。
			resp, err := s.GetPurchases(context.Background(), gen.GetPurchasesRequestObject{Params: gen.GetPurchasesParams{}})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrUnauthenticated)
		})

		t.Run("内部UserIDが未解決のときErrUserIDUnresolvedを返しUsecaseを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			// WithUserID を呼ばず内部 UserID を未解決のまま Authn を載せる（JWT 検証済みだが DB ユーザー未解決の状態）。
			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			resp, err := s.GetPurchases(ctx, gen.GetPurchasesRequestObject{Params: gen.GetPurchasesParams{}})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})

		t.Run("不正なcursorのときErrInvalidArgumentを返しUsecaseを呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			bad := "!!!"
			resp, err := s.GetPurchases(authnContext(t, uuidtestkit.NewTestFromSalt(t, "get_user")), gen.GetPurchasesRequestObject{
				Params: gen.GetPurchasesParams{After: &bad},
			})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("Usecaseのエラーが伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc, idem: idempotency.Deps{}}

			userID := uuidtestkit.NewTestFromSalt(t, "get_user")
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			resp, err := s.GetPurchases(authnContext(t, userID), gen.GetPurchasesRequestObject{Params: gen.GetPurchasesParams{}})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}

func Test_toPeriodSpec(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("期間パラメータが漏れなくユースケースの期間指定へ写像される", func(t *testing.T) {
			t.Parallel()

			kind := gen.GetPurchasesParamsPeriod(period.KindRange)
			from := openapi_types.Date{Time: time.Date(2026, time.January, 21, 0, 0, 0, 0, time.UTC)}
			to := openapi_types.Date{Time: time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)}
			month := "2026-01"
			days := int32(10)

			actual := toPeriodSpec(gen.GetPurchasesParams{
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
			assert.Equal(t, period.Spec{}, toPeriodSpec(gen.GetPurchasesParams{}))
		})
	})
}
