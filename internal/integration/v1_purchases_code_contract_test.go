package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	v1purchases "go-boilerplate/internal/controller/handler/v1/purchases"
	purchasesdetail "go-boilerplate/internal/controller/handler/v1/purchases/detail"
	purchasescancel "go-boilerplate/internal/controller/handler/v1/purchases/detail/cancel"
	purchasesdeliver "go-boilerplate/internal/controller/handler/v1/purchases/detail/deliver"
	purchasespay "go-boilerplate/internal/controller/handler/v1/purchases/detail/pay"
	purchasesship "go-boilerplate/internal/controller/handler/v1/purchases/detail/ship"
	listgen "go-boilerplate/internal/controller/handler/v1/purchases/gen"
	"go-boilerplate/internal/observability"
	authbd "go-boilerplate/internal/usecase/boundary/auth"
	"go-boilerplate/internal/usecase/idempotency"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchaseuc "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// feedCode は、一覧が返す購入コードです。公開識別子が UUID 形式に縛られないことを表すため、
// UUID 形式でない値を選んでいます。
const feedCode = "PC-2026-0042"

// bindPurchaseCodeRoutes は、一覧と詳細系 5 経路を 1 つの Echo に登録します。
func bindPurchaseCodeRoutes(t *testing.T, e *echo.Echo, uc purchaseuc.Usecase) {
	t.Helper()
	tf := observability.NewNoopTracerFactory(t)
	v1purchases.BindHandler(e, tf, uc, nil, idempotency.Deps{})
	purchasesdetail.BindHandler(e, tf, uc)
	purchasescancel.BindHandler(e, tf, uc)
	purchasespay.BindHandler(e, tf, uc)
	purchasesship.BindHandler(e, tf, uc)
	purchasesdeliver.BindHandler(e, tf, uc)
}

// purchaseDetailViewFixture は、詳細系の応答元となるビューを生成するテストヘルパーです。
func purchaseDetailViewFixture(t *testing.T) purchaseuc.PurchaseGetDetailView {
	t.Helper()
	return purchaseuc.PurchaseGetDetailView{
		Code:           feedCode,
		UserID:         uuidtestkit.NewTestFromSalt(t, "code_contract_user"),
		StatusID:       uuidtestkit.NewTestFromSalt(t, "code_contract_status"),
		StatusName:     "支払い済み",
		SubtotalAmount: 160000,
		TaxAmount:      16000,
		ShippingFee:    500,
		TotalAmount:    176500,
		Details: []purchaseuc.PurchaseDetailItemView{
			{
				ProductID:   uuidtestkit.NewTestFromSalt(t, "code_contract_product"),
				ProductName: "ワイヤレスイヤホン",
				Quantity:    2,
				UnitPrice:   decimaltestkit.MustParse(t, "800"),
			},
		},
		OrderedAt: time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC),
	}
}

// purchaseTransitionDetails は、遷移系 4 経路が共有する明細です。
func purchaseTransitionDetails(t *testing.T) []purchaseuc.PurchaseDetailView {
	t.Helper()
	return []purchaseuc.PurchaseDetailView{
		{
			ProductID: uuidtestkit.NewTestFromSalt(t, "code_contract_product"),
			Quantity:  2,
			UnitPrice: decimaltestkit.MustParse(t, "800"),
		},
	}
}

func TestV1PurchasesCodeContract_Integration(t *testing.T) {
	t.Parallel()

	orderedAt := time.Date(2026, time.July, 23, 0, 0, 0, 0, time.UTC)
	at := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	statusID := uuidtestkit.NewTestFromSalt(t, "code_contract_status")
	userID := uuidtestkit.NewTestFromSalt(t, "code_contract_user")

	// listedCode は、一覧を 1 度呼び、その応答が返した購入コードを取り出します。詳細系はこの戻り値
	// だけを入力に取るため、一覧と詳細系が同じ識別子で繋がっていることが呼び出しの形で示されます。
	listedCode := func(t *testing.T, e *echo.Echo, headers http.Header) string {
		t.Helper()
		actual := StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers)
		require.Equal(t, http.StatusOK, actual.StatusCode)

		var body listgen.PurchaseListResponse
		require.NoError(t, json.NewDecoder(actual.Body).Decode(&body))
		require.Len(t, body.Items, 1)
		return body.Items[0].Code
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("一覧が返したcodeだけで詳細・cancel・pay・ship・deliverの5経路すべてを呼べる", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{
					Items: []purchaseuc.PurchaseSummaryView{{
						Code:        feedCode,
						TotalAmount: 176500,
						StatusID:    statusID,
						StatusName:  "支払い済み",
						OrderedAt:   orderedAt,
					}},
				}, nil,
			)

			// 詳細系 5 経路は、一覧が返した code をそのまま受け取ることを各々が検証します。
			uc.EXPECT().GetPurchaseDetail(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *authbd.Authn, code string) (purchaseuc.PurchaseGetDetailView, error) {
					assert.Equal(t, feedCode, code)
					return purchaseDetailViewFixture(t), nil
				},
			)
			uc.EXPECT().CancelPurchase(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params purchaseuc.CancelPurchaseParams) (purchaseuc.CancelPurchaseView, error) {
					assert.Equal(t, feedCode, params.PurchaseCode)
					return purchaseuc.CancelPurchaseView{
						Code: feedCode, UserID: userID, StatusID: statusID, StatusName: "キャンセル",
						TotalAmount: 176500, Details: purchaseTransitionDetails(t), OrderedAt: orderedAt, CanceledAt: &at,
					}, nil
				},
			)
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, params purchaseuc.PayPurchaseParams) (purchaseuc.PayPurchaseView, error) {
					assert.Equal(t, feedCode, params.PurchaseCode)
					return purchaseuc.PayPurchaseView{
						Code: feedCode, UserID: userID, StatusID: statusID, StatusName: "支払い済み",
						TotalAmount: 176500, Details: purchaseTransitionDetails(t), OrderedAt: orderedAt, PaidAt: &at,
					}, nil
				},
			)
			uc.EXPECT().ShipPurchase(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *authbd.Authn, code string) (purchaseuc.ShipPurchaseView, error) {
					assert.Equal(t, feedCode, code)
					return purchaseuc.ShipPurchaseView{
						Code: feedCode, UserID: userID, StatusID: statusID, StatusName: "発送済み",
						TotalAmount: 176500, Details: purchaseTransitionDetails(t), OrderedAt: orderedAt, ShippedAt: &at,
					}, nil
				},
			)
			uc.EXPECT().DeliverPurchase(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ *authbd.Authn, code string) (purchaseuc.DeliverPurchaseView, error) {
					assert.Equal(t, feedCode, code)
					return purchaseuc.DeliverPurchaseView{
						Code: feedCode, UserID: userID, StatusID: statusID, StatusName: "配達済み",
						TotalAmount: 176500, Details: purchaseTransitionDetails(t), OrderedAt: orderedAt, DeliveredAt: &at,
					}, nil
				},
			)

			bindPurchaseCodeRoutes(t, e, uc)
			headers := availablePurchaseUser(t, e)
			code := listedCode(t, e, headers)
			require.Equal(t, feedCode, code)

			for _, tc := range []struct {
				name   string
				method string
				path   string
			}{
				{"詳細", http.MethodGet, "/v1/purchases/" + code},
				{"cancel", http.MethodPatch, "/v1/purchases/" + code + "/cancel"},
				{"pay", http.MethodPatch, "/v1/purchases/" + code + "/pay"},
				{"ship", http.MethodPatch, "/v1/purchases/" + code + "/ship"},
				{"deliver", http.MethodPatch, "/v1/purchases/" + code + "/deliver"},
			} {
				actual := StartServer(t, e).DoJSON(tc.method, tc.path, nil, headers)
				require.Equal(t, http.StatusOK, actual.StatusCode, tc.name)
			}
		})

		t.Run("購入まわりの応答に内部UUIDのidが1つも現れない", func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			ctrl := gomock.NewController(t)

			uc := mock_purchaseuc.NewMockUsecase(ctrl)
			uc.EXPECT().GetPurchases(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
				&purchaseuc.PurchaseListView{
					Items: []purchaseuc.PurchaseSummaryView{{
						Code: feedCode, TotalAmount: 176500, StatusID: statusID, StatusName: "支払い済み", OrderedAt: orderedAt,
					}},
				}, nil,
			)
			uc.EXPECT().GetPurchaseDetail(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(purchaseDetailViewFixture(t), nil)
			uc.EXPECT().PayPurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PayPurchaseView{
				Code: feedCode, UserID: userID, StatusID: statusID, StatusName: "支払い済み",
				TotalAmount: 176500, Details: purchaseTransitionDetails(t), OrderedAt: orderedAt, PaidAt: &at,
			}, nil)

			bindPurchaseCodeRoutes(t, e, uc)
			headers := availablePurchaseUser(t, e)

			// 一覧の各要素と、詳細・遷移系の応答本体に id が無いことを生の JSON で確かめます。
			// 型で消えている以上コンパイルは通るため、契約として固定するには本体を読む必要があります。
			listBody := decodeJSONObject(t, StartServer(t, e).DoJSON(http.MethodGet, "/v1/purchases", nil, headers))
			items, ok := listBody["items"].([]any)
			require.True(t, ok)
			require.Len(t, items, 1)
			assertNoInternalID(t, items[0], "一覧の要素")

			for _, tc := range []struct {
				name   string
				method string
				path   string
			}{
				{"詳細", http.MethodGet, "/v1/purchases/" + feedCode},
				{"pay", http.MethodPatch, "/v1/purchases/" + feedCode + "/pay"},
			} {
				body := decodeJSONObject(t, StartServer(t, e).DoJSON(tc.method, tc.path, nil, headers))
				assertNoInternalID(t, body, tc.name)
			}
		})
	})
}

// decodeJSONObject は、応答本体を型付けせずに JSON オブジェクトとして読み出すテストヘルパーです。
func decodeJSONObject(t *testing.T, res *http.Response) map[string]any {
	t.Helper()
	require.Equal(t, http.StatusOK, res.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
	return body
}

// assertNoInternalID は、購入を表す JSON オブジェクトが内部 UUID の id を持たないことを検証します。
// ステータス参照（status.id）はステータスマスタの識別子であり購入の内部 ID ではないため対象外です。
func assertNoInternalID(t *testing.T, value any, subject string) {
	t.Helper()
	obj, ok := value.(map[string]any)
	require.True(t, ok, subject)
	assert.NotContains(t, obj, "id", "%s は内部 ID を公開してはならない", subject)
	assert.Contains(t, obj, "code", subject)
}
