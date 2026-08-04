package deliver

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/deliver/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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

// deliverViewFixture は、配達完了後の購入ビューを生成するテストヘルパーです。
func deliverViewFixture(t *testing.T) purchaseuc.DeliverPurchaseView {
	t.Helper()
	delivered := time.Date(2026, time.July, 28, 9, 0, 0, 0, time.UTC)
	return purchaseuc.DeliverPurchaseView{
		ID:             uuid.NewTestFromSalt(t, "hd_id"),
		Code:           "hd-code",
		UserID:         uuid.NewTestFromSalt(t, "hd_user"),
		StatusID:       uuid.NewTestFromSalt(t, "hd_status"),
		StatusName:     "配達済み",
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details: []purchaseuc.PurchaseDetailView{
			{ProductID: uuid.NewTestFromSalt(t, "hd_prod"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
		},
		OrderedAt:   time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
		DeliveredAt: &delivered,
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	uc := mock_purchaseuc.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, uc)

	routes := e.Router().Routes()
	require.Len(t, routes, 1)
	assert.Equal(t, http.MethodPatch, routes[0].Method)
	assert.Equal(t, "/v1/purchases/:purchaseId/deliver", routes[0].Path)
}

func Test_server_PatchPurchasesDeliver(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みユーザーの要求で購入を配達済みにし200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuid.NewTestFromSalt(t, "hd_admin")
			purchaseID := uuid.NewTestFromSalt(t, "hd_purchase")
			view := deliverViewFixture(t)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, authn *auth.Authn, id uuid.UUID) (purchaseuc.DeliverPurchaseView, error) {
					// 認可は usecase が行うため、handler は認証主体をそのまま渡す（内部 UserID の解決は行わない）。
					require.NotNil(t, authn)
					resolved, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, resolved)
					assert.Equal(t, purchaseID, id)
					return view, nil
				})

			resp, err := s.PatchPurchasesDeliver(authnContext(t, userID), gen.PatchPurchasesDeliverRequestObject{
				PurchaseId: purchaseID.ToPrimitive(),
			})
			require.NoError(t, err)

			r, ok := resp.(gen.PatchPurchasesDeliver200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, view.ID.ToPrimitive(), r.Id)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "配達済み", r.Status.Name)
			require.Len(t, r.Details, 1)
		})

		t.Run("内部UserIDが未解決でも認証主体を渡してユースケースを呼ぶ", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			// 内部 UserID の解決可否は認可（usecase 側）の判断材料であり、handler は素通しする。
			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			var captured *auth.Authn
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, a *auth.Authn, _ uuid.UUID) (purchaseuc.DeliverPurchaseView, error) {
					captured = a
					return deliverViewFixture(t), nil
				})

			_, err = s.PatchPurchasesDeliver(ctx, gen.PatchPurchasesDeliverRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hd_unresolved").ToPrimitive(),
			})
			require.NoError(t, err)

			// 内部 UserID が未解決でも認証主体そのものは usecase へ渡る（解決可否の判断は認可側の責務）。
			require.NotNil(t, captured)
			_, uerr := captured.UserID()
			require.ErrorIs(t, uerr, auth.ErrUserIDUnresolved)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が無い場合、ErrUnauthenticatedUserを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.PatchPurchasesDeliver(context.Background(), gen.PatchPurchasesDeliverRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hd_noauth").ToPrimitive(),
			})
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("非adminの認可エラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.DeliverPurchaseView{}, apperror.ErrPermissionDenied)

			userID := uuid.NewTestFromSalt(t, "hd_user_forbidden")
			_, err := s.PatchPurchasesDeliver(authnContext(t, userID), gen.PatchPurchasesDeliverRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hd_purchase_forbidden").ToPrimitive(),
			})
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("ユースケースが不正遷移エラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.DeliverPurchaseView{}, apperror.ErrConflict)

			userID := uuid.NewTestFromSalt(t, "hd_user_err")
			_, err := s.PatchPurchasesDeliver(authnContext(t, userID), gen.PatchPurchasesDeliverRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hd_purchase_err").ToPrimitive(),
			})
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("レスポンス変換が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			view := deliverViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).Return(view, nil)

			userID := uuid.NewTestFromSalt(t, "hd_user_overflow")
			_, err := s.PatchPurchasesDeliver(authnContext(t, userID), gen.PatchPurchasesDeliverRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hd_purchase_overflow").ToPrimitive(),
			})
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toDeliverResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("配達ビューをステータス参照付きレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := deliverViewFixture(t)
			r, err := toDeliverResponse(view)
			require.NoError(t, err)
			assert.Equal(t, view.ID.ToPrimitive(), r.Id)
			assert.Equal(t, view.Code, r.Code)
			assert.Equal(t, view.UserID.ToPrimitive(), r.UserId)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "配達済み", r.Status.Name)
			assert.Equal(t, int64(176500), r.TotalAmount)
			assert.Equal(t, *view.DeliveredAt, r.DeliveredAt)
			require.Len(t, r.Details, 1)
			assert.Equal(t, view.Details[0].ProductID.ToPrimitive(), r.Details[0].ProductId)
		})

		t.Run("配達日時がnilの場合はゼロ値へ倒す", func(t *testing.T) {
			t.Parallel()

			view := deliverViewFixture(t)
			view.DeliveredAt = nil
			r, err := toDeliverResponse(view)
			require.NoError(t, err)
			assert.Zero(t, r.DeliveredAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量がint32範囲を超える場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			view := deliverViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			_, err := toDeliverResponse(view)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}
