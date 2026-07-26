package ship

import (
	"context"
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
	"go-boilerplate/pkg/uuid"

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
	require.True(t, ctxhelper.SetAuthn(ctx, *authn.WithUserID(userID)))
	return ctx
}

// shipViewFixture は、発送後の購入ビューを生成するテストヘルパーです。
func shipViewFixture(t *testing.T) purchaseuc.ShipPurchaseView {
	t.Helper()
	shipped := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	return purchaseuc.ShipPurchaseView{
		ID:             uuid.NewTestFromSalt(t, "hs_id"),
		Code:           "hs-code",
		UserID:         uuid.NewTestFromSalt(t, "hs_user"),
		StatusID:       uuid.NewTestFromSalt(t, "hs_status"),
		StatusName:     "発送済み",
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details: []purchaseuc.PurchaseDetailView{
			{ProductID: uuid.NewTestFromSalt(t, "hs_prod"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
		},
		OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
		ShippedAt: &shipped,
	}
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

			userID := uuid.NewTestFromSalt(t, "hs_admin")
			purchaseID := uuid.NewTestFromSalt(t, "hs_purchase")
			view := shipViewFixture(t)
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, authn *auth.Authn, id uuid.UUID) (purchaseuc.ShipPurchaseView, error) {
					// 認可は usecase が行うため、handler は認証主体をそのまま渡す（内部 UserID の解決は行わない）。
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

			// 内部 UserID の解決可否は認可（usecase 側）の判断材料であり、handler は素通しする。
			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(shipViewFixture(t), nil)

			_, err = s.PatchPurchasesShip(ctx, gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hs_unresolved").ToPrimitive(),
			})
			require.NoError(t, err)
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
				PurchaseId: uuid.NewTestFromSalt(t, "hs_noauth").ToPrimitive(),
			})
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("非adminの認可エラーをそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.ShipPurchaseView{}, apperror.ErrPermissionDenied)

			userID := uuid.NewTestFromSalt(t, "hs_user_forbidden")
			_, err := s.PatchPurchasesShip(authnContext(t, userID), gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hs_purchase_forbidden").ToPrimitive(),
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

			userID := uuid.NewTestFromSalt(t, "hs_user_err")
			_, err := s.PatchPurchasesShip(authnContext(t, userID), gen.PatchPurchasesShipRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hs_purchase_err").ToPrimitive(),
			})
			require.ErrorIs(t, err, apperror.ErrConflict)
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
			r := toShipResponse(view)
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
	})
}

func Test_shippedAt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("非nilの場合は値を返す", func(t *testing.T) {
			t.Parallel()
			tm := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
			assert.Equal(t, tm, shippedAt(&tm))
		})

		t.Run("nilの場合はゼロ値を返す", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, time.Time{}, shippedAt(nil))
		})
	})
}
