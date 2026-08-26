package cancel

import (
	"context"
	"math"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/v1/purchases/detail/cancel/gen"
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

// cancelViewFixture は、キャンセル後の購入ビューを生成するテストヘルパーです。
func cancelViewFixture(t *testing.T) purchaseuc.CancelPurchaseView {
	t.Helper()
	canceledAt := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	return purchaseuc.CancelPurchaseView{
		Code:           "hc-code",
		UserID:         uuidtestkit.NewTestFromSalt(t, "hc_user"),
		StatusID:       uuidtestkit.NewTestFromSalt(t, "hc_status"),
		StatusCode:     6,
		StatusName:     "キャンセル",
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details: []purchaseuc.PurchaseDetailView{
			{ProductID: uuidtestkit.NewTestFromSalt(t, "hc_prod"), Quantity: 2, UnitPrice: decimaltestkit.MustParse(t, "800")},
		},
		OrderedAt:  time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
		CanceledAt: &canceledAt,
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
	assert.Equal(t, "/v1/purchases/:purchaseCode/cancel", routes[0].Path)
}

func Test_server_PatchPurchasesCancel(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みユーザーの購入をキャンセルし200で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			userID := uuidtestkit.NewTestFromSalt(t, "hc_user")
			const purchaseCode = "hc-code"
			view := cancelViewFixture(t)
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, params purchaseuc.CancelPurchaseParams) (purchaseuc.CancelPurchaseView, error) {
					assert.Equal(t, userID, params.UserID)
					assert.Equal(t, purchaseCode, params.PurchaseCode)
					return view, nil
				})

			resp, err := s.PatchPurchasesCancel(authnContext(t, userID), gen.PatchPurchasesCancelRequestObject{
				PurchaseCode: purchaseCode,
			})
			require.NoError(t, err)

			r, ok := resp.(gen.PatchPurchasesCancel200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, view.Code, r.Code)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "キャンセル", r.Status.Name)
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

			_, err := s.PatchPurchasesCancel(context.Background(), gen.PatchPurchasesCancelRequestObject{
				PurchaseCode: "hc-noauth",
			})
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("ユースケースがエラーを返した場合はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).
				Return(purchaseuc.CancelPurchaseView{}, apperror.ErrNotFound)

			userID := uuidtestkit.NewTestFromSalt(t, "hc_user_err")
			_, err := s.PatchPurchasesCancel(authnContext(t, userID), gen.PatchPurchasesCancelRequestObject{
				PurchaseCode: "hc-purchase-err",
			})
			require.ErrorIs(t, err, apperror.ErrNotFound)
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

			_, err = s.PatchPurchasesCancel(ctx, gen.PatchPurchasesCancelRequestObject{
				PurchaseCode: "hc-unresolved",
			})
			require.ErrorIs(t, err, auth.ErrUserIDUnresolved)
		})

		t.Run("レスポンス変換が失敗した場合はエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			s := &server{tracer: observability.NewMockControllerLayerTracer(t), uc: uc}

			view := cancelViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).Return(view, nil)

			userID := uuidtestkit.NewTestFromSalt(t, "hc_user_overflow")
			_, err := s.PatchPurchasesCancel(authnContext(t, userID), gen.PatchPurchasesCancelRequestObject{
				PurchaseCode: "hc-purchase-overflow",
			})
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}

func Test_toCancelResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセルビューをステータス参照付きレスポンスへ写像する", func(t *testing.T) {
			t.Parallel()

			view := cancelViewFixture(t)
			r, err := toCancelResponse(view)
			require.NoError(t, err)
			assert.Equal(t, view.Code, r.Code)
			assert.Equal(t, view.UserID.ToPrimitive(), r.UserId)
			assert.Equal(t, view.StatusID.ToPrimitive(), r.Status.Id)
			assert.Equal(t, "キャンセル", r.Status.Name)
			assert.EqualValues(t, view.StatusCode, r.Status.Code)
			assert.Equal(t, int64(176500), r.TotalAmount)
			assert.Equal(t, *view.CanceledAt, r.CanceledAt)
			require.Len(t, r.Details, 1)
			assert.Equal(t, view.Details[0].ProductID.ToPrimitive(), r.Details[0].ProductId)
		})

		t.Run("キャンセル日時がnilの場合はゼロ値へ倒す", func(t *testing.T) {
			t.Parallel()

			view := cancelViewFixture(t)
			view.CanceledAt = nil
			r, err := toCancelResponse(view)
			require.NoError(t, err)
			assert.Zero(t, r.CanceledAt)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("数量がint32範囲を超える場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			view := cancelViewFixture(t)
			view.Details[0].Quantity = math.MaxInt32 + 1
			_, err := toCancelResponse(view)
			require.ErrorIs(t, err, safecast.ErrOverflow)
		})
	})
}
