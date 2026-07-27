package pay

import (
	"context"
	"math"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/pay/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	"go-boilerplate/pkg/safecast"
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

// payViewFixture は、支払い後の購入ビューを生成するテストヘルパーです。
func payViewFixture(t *testing.T) purchaseuc.PayPurchaseView {
	t.Helper()
	paidAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	return purchaseuc.PayPurchaseView{
		ID:             uuid.NewTestFromSalt(t, "hp_id"),
		Code:           "hp-code",
		UserID:         uuid.NewTestFromSalt(t, "hp_user"),
		StatusID:       uuid.NewTestFromSalt(t, "hp_status"),
		StatusName:     "支払い済み",
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details: []purchaseuc.PurchaseDetailView{
			{ProductID: uuid.NewTestFromSalt(t, "hp_prod"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
		},
		OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
		PaidAt:    &paidAt,
	}
}

func Test_server_PatchPurchasesPay(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みユーザーの購入を支払い済みにし200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuid.NewTestFromSalt(t, "hp_user")
			purchaseID := uuid.NewTestFromSalt(t, "hp_purchase")
			view := payViewFixture(t)
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, params purchaseuc.PayPurchaseParams) (purchaseuc.PayPurchaseView, error) {
					assert.Equal(t, userID, params.UserID)
					assert.Equal(t, purchaseID, params.PurchaseID)
					return view, nil
				})

			resp, err := s.PatchPurchasesPay(authnContext(t, userID), gen.PatchPurchasesPayRequestObject{
				PurchaseId: purchaseID.ToPrimitive(),
			})
			require.NoError(t, err)

			r, ok := resp.(gen.PatchPurchasesPay200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, view.ID.ToPrimitive(), r.Id)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "支払い済み", r.Status.Name)
			require.Len(t, r.Details, 1)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が無い場合、ErrUnauthenticatedUserを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			_, err := s.PatchPurchasesPay(context.Background(), gen.PatchPurchasesPayRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hp_noauth").ToPrimitive(),
			})
			require.ErrorIs(t, err, ErrUnauthenticatedUser)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).
				Return(purchaseuc.PayPurchaseView{}, apperror.ErrConflict)

			userID := uuid.NewTestFromSalt(t, "hp_user_err")
			_, err := s.PatchPurchasesPay(authnContext(t, userID), gen.PatchPurchasesPayRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hp_purchase_err").ToPrimitive(),
			})
			require.ErrorIs(t, err, apperror.ErrConflict)
		})

		t.Run("内部UserIDが未解決の場合、エラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			// WithUserID を呼ばず内部 UserID を未解決のまま載せる。
			ctx := ctxhelper.WithAuthn(context.Background())
			authn, err := auth.New("subject", "issuer", nil, nil)
			require.NoError(t, err)
			require.True(t, ctxhelper.SetAuthn(ctx, *authn))

			_, err = s.PatchPurchasesPay(ctx, gen.PatchPurchasesPayRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hp_unresolved").ToPrimitive(),
			})
			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})

		t.Run("レスポンス変換が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			view := payViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).Return(view, nil)

			userID := uuid.NewTestFromSalt(t, "hp_user_overflow")
			_, err := s.PatchPurchasesPay(authnContext(t, userID), gen.PatchPurchasesPayRequestObject{
				PurchaseId: uuid.NewTestFromSalt(t, "hp_purchase_overflow").ToPrimitive(),
			})
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toPayResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("支払いビューをステータス参照付きレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := payViewFixture(t)
			r, err := toPayResponse(view)
			require.NoError(t, err)
			assert.Equal(t, view.ID.ToPrimitive(), r.Id)
			assert.Equal(t, view.Code, r.Code)
			assert.Equal(t, view.UserID.ToPrimitive(), r.UserId)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "支払い済み", r.Status.Name)
			assert.Equal(t, int64(176500), r.TotalAmount)
			assert.Equal(t, *view.PaidAt, r.PaidAt)
			require.Len(t, r.Details, 1)
			assert.Equal(t, view.Details[0].ProductID.ToPrimitive(), r.Details[0].ProductId)
		})

		t.Run("支払い日時がnilの場合はゼロ値へ倒す", func(t *testing.T) {
			t.Parallel()

			view := payViewFixture(t)
			view.PaidAt = nil
			r, err := toPayResponse(view)
			require.NoError(t, err)
			assert.Zero(t, r.PaidAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量がint32範囲を超える場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			view := payViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			_, err := toPayResponse(view)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}
