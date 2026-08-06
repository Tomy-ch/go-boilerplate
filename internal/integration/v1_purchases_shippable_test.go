package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/apperror"
	purchasesshippablehandler "go-boilerplate/internal/controller/handler/v1/purchases/shippable"
	"go-boilerplate/internal/controller/handler/v1/purchases/shippable/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/auth"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchase "go-boilerplate/internal/usecase/purchase/mock"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const purchasesShippablePath = "/v1/purchases/shippable"

//nolint:dupl // 在庫僅少/発送待ちはどちらも admin 向け top-N 一覧で、HTTP 写像テストの構造の重複は不可避
func TestV1PurchasesShippable_Integration(t *testing.T) {
	t.Parallel()

	// shippableView は、購入者ごとに分かれた 2 組（複数件の組と単独の組）を返します。
	shippableView := func(t *testing.T) purchaseuc.PurchaseShippableListView {
		t.Helper()
		base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		newItem := func(salt string, orderedAt time.Time) purchaseuc.ShippablePurchaseView {
			return purchaseuc.ShippablePurchaseView{
				ID:          uuidtestkit.NewTestFromSalt(t, salt),
				Code:        "code-" + salt,
				TotalAmount: 176500,
				OrderedAt:   orderedAt,
			}
		}
		return purchaseuc.PurchaseShippableListView{
			Groups: []purchaseuc.DispatchGroupView{
				{
					UserID: uuidtestkit.NewTestFromSalt(t, "int_shippable_alice"),
					Purchases: []purchaseuc.ShippablePurchaseView{
						newItem("int_shippable_a1", base),
						newItem("int_shippable_a2", base.Add(2*time.Hour)),
					},
				},
				{
					UserID: uuidtestkit.NewTestFromSalt(t, "int_shippable_bob"),
					Purchases: []purchaseuc.ShippablePurchaseView{
						newItem("int_shippable_b1", base.Add(time.Hour)),
					},
				},
			},
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("認証済みでまとめ発送一覧がPurchaseShippableResponseで返る", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchase.NewMockUsecase(ctrl)
			uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(shippableView(t), nil)

			purchasesshippablehandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_shippable_admin"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasesShippablePath, nil, headers)
			assert.Equal(t, http.StatusOK, actual.StatusCode)
			AssertJSONResponseType[gen.PurchaseShippableResponse](t, actual)
		})

		t.Run("limitがユースケースへ伝わる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			var captured purchaseuc.ListShippablePurchasesParams
			uc := mock_purchase.NewMockUsecase(ctrl)
			uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(
					_ context.Context, _ *auth.Authn, params purchaseuc.ListShippablePurchasesParams,
				) (purchaseuc.PurchaseShippableListView, error) {
					captured = params
					return shippableView(t), nil
				})

			purchasesshippablehandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_shippable_limit"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasesShippablePath+"?limit=5", nil, headers)
			require.Equal(t, http.StatusOK, actual.StatusCode)
			assert.Equal(t, 5, captured.Limit)
		})

		t.Run("OpenAPIバリデーション経由でも範囲内のlimitは200で通過する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchase.NewMockUsecase(ctrl)
			uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(shippableView(t), nil)

			purchasesshippablehandler.BindHandler(e, tf, uc)
			useOpenAPIValidation(t, e)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_shippable_valid"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasesShippablePath+"?limit=100", nil, headers)
			AssertJSONResponseType[gen.PurchaseShippableResponse](t, actual)
		})

		t.Run("発送待ちの購入が無い場合は空配列を200で返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchase.NewMockUsecase(ctrl)
			uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.PurchaseShippableListView{}, nil)

			purchasesshippablehandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_shippable_empty"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasesShippablePath, nil, headers)
			// 対象が空でも groups は null ではなく [] でシリアライズされる（AssertJSONResponseType が検査）。
			AssertJSONResponseType[gen.PurchaseShippableResponse](t, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未認証で401を返しUsecaseは呼ばれない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchase.NewMockUsecase(ctrl)
			uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			purchasesshippablehandler.BindHandler(e, tf, uc)

			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasesShippablePath, nil, nil)
			AssertErrorResponse(t, actual, http.StatusUnauthorized)
		})

		t.Run("非adminの権限エラーを403へ変換する", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchase.NewMockUsecase(ctrl)
			uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.PurchaseShippableListView{}, apperror.ErrPermissionDenied)

			purchasesshippablehandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_shippable_forbidden"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasesShippablePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusForbidden)
		})

		t.Run("OpenAPIバリデーションが範囲外のlimitを400で弾く", func(t *testing.T) {
			t.Parallel()

			t.Run("limitが下限未満(0)", func(t *testing.T) {
				t.Parallel()
				assertShippableLimitRejected(t, purchasesShippablePath+"?limit=0")
			})

			t.Run("limitが上限超過(101)", func(t *testing.T) {
				t.Parallel()
				assertShippableLimitRejected(t, purchasesShippablePath+"?limit=101")
			})
		})

		t.Run("ErrInternalで500を返す", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			UseAppErrorHandler(t, e)
			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)

			uc := mock_purchase.NewMockUsecase(ctrl)
			uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseuc.PurchaseShippableListView{}, apperror.ErrInternal)

			purchasesshippablehandler.BindHandler(e, tf, uc)

			headers := MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_shippable_err"))
			actual := StartServer(t, e).DoJSON(http.MethodGet, purchasesShippablePath, nil, headers)
			AssertErrorResponse(t, actual, http.StatusInternalServerError)
		})
	})
}

// assertShippableLimitRejected は、OpenAPI バリデータが limit を 400 で弾き Usecase へ到達しないことを検証します。
func assertShippableLimitRejected(t *testing.T, path string) {
	t.Helper()

	e := echo.New()
	UseAppErrorHandler(t, e)
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	uc := mock_purchase.NewMockUsecase(ctrl)
	uc.EXPECT().ListShippablePurchases(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	purchasesshippablehandler.BindHandler(e, tf, uc)
	useOpenAPIValidation(t, e)

	actual := StartServer(t, e).DoJSON(
		http.MethodGet, path, nil, MakeAvailableUserID(t, e, uuidtestkit.NewTestFromSalt(t, "int_shippable_limit_ng")),
	)
	AssertErrorResponse(t, actual, http.StatusBadRequest)
}
