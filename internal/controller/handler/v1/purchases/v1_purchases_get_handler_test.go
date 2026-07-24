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
	"go-boilerplate/internal/usecase/tools/paging"
	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestSummaryView() purchaseuc.PurchaseSummaryView {
	return purchaseuc.PurchaseSummaryView{
		Code:        "h-code",
		TotalAmount: 176500,
		Status:      "完了",
		OrderedAt:   time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
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

			userID := uuid.NewTestFromSalt(t, "get_user")
			view := newTestSummaryView()
			nextCursor := "next-opaque-cursor"
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, uid uuid.UUID, _ *paging.Cursor) (*purchaseuc.PurchaseListView, error) {
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
					Status:      "完了",
					OrderedAt:   view.OrderedAt,
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

			userID := uuid.NewTestFromSalt(t, "get_user")
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).
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

			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

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

			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

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

			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			bad := "!!!"
			resp, err := s.GetPurchases(authnContext(t, uuid.NewTestFromSalt(t, "get_user")), gen.GetPurchasesRequestObject{
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

			userID := uuid.NewTestFromSalt(t, "get_user")
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, apperror.ErrInternal)

			resp, err := s.GetPurchases(authnContext(t, userID), gen.GetPurchasesRequestObject{Params: gen.GetPurchasesParams{}})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrInternal)
		})
	})
}
