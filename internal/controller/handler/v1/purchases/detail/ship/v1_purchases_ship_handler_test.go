package ship

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/ship/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/safecast"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

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

// shipViewFixture は、発送後の購入ビューを生成するテストヘルパーです。
func shipViewFixture(t *testing.T) purchaseuc.ShipPurchaseView {
	t.Helper()
	shipped := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	return purchaseuc.ShipPurchaseView{
		ID:             uuidtestkit.NewTestFromSalt(t, "hs_id"),
		Code:           "hs-code",
		UserID:         uuidtestkit.NewTestFromSalt(t, "hs_user"),
		StatusID:       uuidtestkit.NewTestFromSalt(t, "hs_status"),
		StatusName:     "発送済み",
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details: []purchaseuc.PurchaseDetailView{
			{ProductID: uuidtestkit.NewTestFromSalt(t, "hs_prod"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
		},
		OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
		ShippedAt: &shipped,
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
	assert.Equal(t, "/v1/purchases/:purchaseId/ship", routes[0].Path)
}

func Test_server_PatchPurchasesShip(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みユーザーの要求で購入を発送済みにし200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "hs_admin")
			purchaseID := uuidtestkit.NewTestFromSalt(t, "hs_purchase")
			view := shipViewFixture(t)
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, authn *auth.Authn, id uuid.UUID) (purchaseuc.ShipPurchaseView, error) {
					require.NotNil(t, authn)
					resolved, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, resolved)
					assert.Equal(t, purchaseID, id)
					return view, nil
				})

			resp, err := s.PatchPurchasesShip(authnContext(t, userID), gen.PatchPurchasesShipRequestObject{
				PurchaseId: purchaseID.ToPrimitive(),
			})
			require.NoError(t, err)

			r, ok := resp.(gen.PatchPurchasesShip200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, view.ID.ToPrimitive(), r.Id)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "発送済み", r.Status.Name)
			require.Len(t, r.Details, 1)
		})

		t.Run("内部UserIDが未解決でも認証主体を渡してユースケースを呼ぶ", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			var captured *auth.Authn
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, a *auth.Authn, _ uuid.UUID) (purchaseuc.ShipPurchaseView, error) {
					captured = a
					return shipViewFixture(t), nil
				})

			_, err = s.PatchPurchasesShip(ctx, gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuidtestkit.NewTestFromSalt(t, "hs_unresolved").ToPrimitive(),
			})
			require.NoError(t, err)

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

			_, err := s.PatchPurchasesShip(context.Background(), gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuidtestkit.NewTestFromSalt(t, "hs_noauth").ToPrimitive(),
			})
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("非adminの認可エラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.ShipPurchaseView{}, apperror.ErrPermissionDenied)

			userID := uuidtestkit.NewTestFromSalt(t, "hs_user_forbidden")
			_, err := s.PatchPurchasesShip(authnContext(t, userID), gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuidtestkit.NewTestFromSalt(t, "hs_purchase_forbidden").ToPrimitive(),
			})
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})

		t.Run("ユースケースが不正遷移エラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.ShipPurchaseView{}, apperror.ErrConflict)

			userID := uuidtestkit.NewTestFromSalt(t, "hs_user_err")
			_, err := s.PatchPurchasesShip(authnContext(t, userID), gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuidtestkit.NewTestFromSalt(t, "hs_purchase_err").ToPrimitive(),
			})
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("レスポンス変換が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			view := shipViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).Return(view, nil)

			userID := uuidtestkit.NewTestFromSalt(t, "hs_user_overflow")
			_, err := s.PatchPurchasesShip(authnContext(t, userID), gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuidtestkit.NewTestFromSalt(t, "hs_purchase_overflow").ToPrimitive(),
			})
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toShipResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("発送ビューをステータス参照付きレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := shipViewFixture(t)
			r, err := toShipResponse(view)
			require.NoError(t, err)
			assert.Equal(t, view.ID.ToPrimitive(), r.Id)
			assert.Equal(t, view.Code, r.Code)
			assert.Equal(t, view.UserID.ToPrimitive(), r.UserId)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "発送済み", r.Status.Name)
			assert.Equal(t, int64(176500), r.TotalAmount)
			assert.Equal(t, *view.ShippedAt, r.ShippedAt)
			require.Len(t, r.Details, 1)
			assert.Equal(t, view.Details[0].ProductID.ToPrimitive(), r.Details[0].ProductId)
		})

		t.Run("発送日時がnilの場合はゼロ値へ倒す", func(t *testing.T) {
			t.Parallel()

			view := shipViewFixture(t)
			view.ShippedAt = nil
			r, err := toShipResponse(view)
			require.NoError(t, err)
			assert.Zero(t, r.ShippedAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量がint32範囲を超える場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			view := shipViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			_, err := toShipResponse(view)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}
