package shippable

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/controller/ctxhelper"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/v1/purchases/shippable/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchase "go-boilerplate/internal/usecase/purchase/mock"
	"go-boilerplate/pkg/ptr"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/v1/purchases/shippable"

func newServer(t *testing.T) (*server, *mock_purchase.MockUsecase) {
	t.Helper()
	mockUC := mock_purchase.NewMockUsecase(gomock.NewController(t))
	return &server{tracer: observability.NewMockControllerLayerTracer(t), uc: mockUC}, mockUC
}

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

func newDispatchGroupView(t *testing.T, salt string, purchaseSalts ...string) purchaseuc.DispatchGroupView {
	t.Helper()

	purchases := make([]purchaseuc.ShippablePurchaseView, len(purchaseSalts))
	for i, ps := range purchaseSalts {
		purchases[i] = purchaseuc.ShippablePurchaseView{
			Code:        "code-" + ps,
			TotalAmount: 1000 * (i + 1),
			OrderedAt:   time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC).Add(time.Duration(i) * time.Hour),
		}
	}
	return purchaseuc.DispatchGroupView{
		UserID:    uuidtestkit.NewTestFromSalt(t, salt),
		Purchases: purchases,
	}
}

// wantDispatchGroupResponse は、本番 toDispatchGroupResponse とは独立な検証用オラクル（フィールド取り違え検出）。
func wantDispatchGroupResponse(dto purchaseuc.DispatchGroupView) gen.PurchaseDispatchGroupResponse {
	purchases := make([]gen.PurchaseShippableItemResponse, len(dto.Purchases))
	for i, p := range dto.Purchases {
		purchases[i] = gen.PurchaseShippableItemResponse{
			Code:        p.Code,
			TotalAmount: int64(p.TotalAmount),
			OrderedAt:   p.OrderedAt,
		}
	}
	return gen.PurchaseDispatchGroupResponse{
		UserId:    dto.UserID.ToPrimitive(),
		Purchases: purchases,
	}
}

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)
	mockUC := mock_purchase.NewMockUsecase(gomock.NewController(t))

	BindHandler(e, tf, mockUC)

	testassert.AssertEchoRouterPath(t, targetPath, e.Router().Routes())
	testassert.AssertEchoRouterMethods(t, []string{http.MethodGet}, e.Router().Routes())
}

func Test_limitParam(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指定された件数をそのまま返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 7, limitParam(ptr.To(7)))
		})

		t.Run("未指定の場合、既定件数を意味する0を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 0, limitParam(nil))
		})
	})
}

func Test_toDispatchGroupResponse(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("組の購入者と購入一覧を順序どおり写像する", func(t *testing.T) {
			t.Parallel()

			dto := newDispatchGroupView(t, "grp_user", "grp_p1", "grp_p2")

			assert.Equal(t, wantDispatchGroupResponse(dto), toDispatchGroupResponse(dto))
		})

		t.Run("購入が空の組は空配列の購入一覧になる", func(t *testing.T) {
			t.Parallel()

			dto := newDispatchGroupView(t, "grp_empty_user")

			assert.Empty(t, toDispatchGroupResponse(dto).Purchases)
		})
	})
}

func Test_server_GetPurchasesShippable(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ユースケースの組を順序どおりPurchaseShippableResponseへ写像する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			first := newDispatchGroupView(t, "h_alice", "h_a1", "h_a2")
			second := newDispatchGroupView(t, "h_bob", "h_b1")
			mockUC.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.PurchaseShippableListView{
					Groups: []purchaseuc.DispatchGroupView{first, second},
				}, nil)

			resp, err := s.GetPurchasesShippable(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "h_admin")),
				gen.GetPurchasesShippableRequestObject{
					Params: gen.GetPurchasesShippableParams{Limit: ptr.To(20)},
				},
			)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPurchasesShippable200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, gen.PurchaseShippableResponse{
				Groups: []gen.PurchaseDispatchGroupResponse{
					wantDispatchGroupResponse(first),
					wantDispatchGroupResponse(second),
				},
			}, gen.PurchaseShippableResponse(actual))
		})

		t.Run("limitとAuthnがユースケースへ引き渡される", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			userID := uuidtestkit.NewTestFromSalt(t, "h_limit_admin")

			var captured purchaseuc.ListShippablePurchasesParams
			mockUC.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, authn *auth.Authn, params purchaseuc.ListShippablePurchasesParams,
				) (purchaseuc.PurchaseShippableListView, error) {
					captured = params
					uid, uerr := authn.UserID()
					require.NoError(t, uerr)
					assert.Equal(t, userID, uid)
					return purchaseuc.PurchaseShippableListView{}, nil
				})

			_, err := s.GetPurchasesShippable(
				authnContext(t, userID),
				gen.GetPurchasesShippableRequestObject{
					Params: gen.GetPurchasesShippableParams{Limit: ptr.To(7)},
				},
			)
			require.NoError(t, err)
			assert.Equal(t, 7, captured.Limit)
		})

		t.Run("limit未指定の場合、既定件数を意味する0がユースケースへ渡る", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)

			var captured purchaseuc.ListShippablePurchasesParams
			mockUC.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, _ *auth.Authn, params purchaseuc.ListShippablePurchasesParams,
				) (purchaseuc.PurchaseShippableListView, error) {
					captured = params
					return purchaseuc.PurchaseShippableListView{}, nil
				})

			_, err := s.GetPurchasesShippable(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "h_nolimit_admin")),
				gen.GetPurchasesShippableRequestObject{Params: gen.GetPurchasesShippableParams{}},
			)
			require.NoError(t, err)
			assert.Equal(t, 0, captured.Limit)
		})

		t.Run("発送待ちの購入が無い場合、空配列のレスポンスが返る", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.PurchaseShippableListView{}, nil)

			resp, err := s.GetPurchasesShippable(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "h_empty_admin")),
				gen.GetPurchasesShippableRequestObject{},
			)
			require.NoError(t, err)

			actual, ok := resp.(gen.GetPurchasesShippable200JSONResponse)
			require.True(t, ok)
			assert.Empty(t, actual.Groups)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証情報が取得できない場合、ErrUnauthenticatedUserを返しユースケースを呼ばない", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			resp, err := s.GetPurchasesShippable(context.Background(), gen.GetPurchasesShippableRequestObject{})
			assert.Nil(t, resp)
			require.ErrorIs(t, err, ctxhelper.ErrUnauthenticatedUser)
		})

		t.Run("ユースケースが権限エラーを返した場合、そのまま伝播する", func(t *testing.T) {
			t.Parallel()

			s, mockUC := newServer(t)
			mockUC.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.PurchaseShippableListView{}, apperror.ErrPermissionDenied)

			resp, err := s.GetPurchasesShippable(
				authnContext(t, uuidtestkit.NewTestFromSalt(t, "h_forbidden_admin")),
				gen.GetPurchasesShippableRequestObject{},
			)
			assert.Nil(t, resp)
			require.ErrorIs(t, err, apperror.ErrPermissionDenied)
		})
	})
}
