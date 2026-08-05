package checkout

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/exchangerate"
	mock_exchangerate "go-boilerplate/internal/usecase/exchangerate/mock"
	purchaseuc "go-boilerplate/internal/usecase/purchase"
	mock_purchase "go-boilerplate/internal/usecase/purchase/mock"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"
	"go-boilerplate/pkg/xerrors"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を保持したユースケースを生成する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			tf := observability.NewNoopTracerFactory(t)
			purchase := mock_purchase.NewMockUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)

			expected := &usecase{tracer: tf.Usecase(), purchase: purchase, xr: xr}
			assert.Equal(t, expected, New(purchase, xr, tf))
		})
	})
}

func Test_usecase_CreatePurchase(t *testing.T) {
	t.Parallel()

	userID := uuidtestkit.NewTestFromSalt(t, "checkout_user")
	created := purchaseuc.PurchaseView{TotalAmount: 10000}

	newUsecase := func(t *testing.T, purchase *mock_purchase.MockUsecase, xr *mock_exchangerate.MockUsecase) *usecase {
		t.Helper()
		return &usecase{
			tracer:   observability.NewNoopTracerFactory(t).Usecase(),
			purchase: purchase,
			xr:       xr,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("表示通貨が未指定の場合、購入のみを返し為替は呼ばない", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			purchase := mock_purchase.NewMockUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			purchase.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(created, nil)

			view, err := newUsecase(t, purchase, xr).CreatePurchase(
				context.Background(), CreatePurchaseParams{UserID: userID},
			)
			require.NoError(t, err)
			assert.Equal(t, created, view.Purchase)
			assert.Nil(t, view.ReferenceAmount)
		})

		t.Run("表示通貨が指定された場合、参考換算額を添えて返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			purchase := mock_purchase.NewMockUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			purchase.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(created, nil)
			xr.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(&exchangerate.ConvertResult{
				Reference: &exchangerate.ReferenceAmount{
					Currency: "JPY",
					Amount:   15050,
					Rate:     decimaltestkit.MustParse(t, "150.5"),
					RateDate: "2026-07-21",
				},
			}, nil)

			jpy := "JPY"
			view, err := newUsecase(t, purchase, xr).CreatePurchase(
				context.Background(), CreatePurchaseParams{UserID: userID, DisplayCurrency: &jpy},
			)
			require.NoError(t, err)
			require.NotNil(t, view.ReferenceAmount)
			assert.Equal(t, "JPY", view.ReferenceAmount.Currency)
			assert.Equal(t, int64(15050), view.ReferenceAmount.Amount)
			assert.Equal(t, "150.5", view.ReferenceAmount.Rate.String())
			assert.Equal(t, "2026-07-21", view.ReferenceAmount.RateDate)
		})

		t.Run("為替が失敗しても購入は成立し参考換算額のみnilになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			purchase := mock_purchase.NewMockUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			purchase.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(created, nil)
			xr.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(nil, xerrors.New("gateway down"))

			jpy := "JPY"
			view, err := newUsecase(t, purchase, xr).CreatePurchase(
				context.Background(), CreatePurchaseParams{UserID: userID, DisplayCurrency: &jpy},
			)
			require.NoError(t, err)
			assert.Equal(t, created, view.Purchase)
			assert.Nil(t, view.ReferenceAmount)
		})

		t.Run("参考換算額が返らない場合はnilになる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			purchase := mock_purchase.NewMockUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			purchase.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(created, nil)
			xr.EXPECT().Convert(gomock.Any(), gomock.Any()).Return(&exchangerate.ConvertResult{Reference: nil}, nil)

			jpy := "JPY"
			view, err := newUsecase(t, purchase, xr).CreatePurchase(
				context.Background(), CreatePurchaseParams{UserID: userID, DisplayCurrency: &jpy},
			)
			require.NoError(t, err)
			assert.Nil(t, view.ReferenceAmount)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("購入の作成が失敗した場合、為替を呼ばずエラーを伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			purchase := mock_purchase.NewMockUsecase(ctrl)
			xr := mock_exchangerate.NewMockUsecase(ctrl)
			expected := xerrors.New("create failed")
			purchase.EXPECT().CreatePurchase(gomock.Any(), gomock.Any()).Return(purchaseuc.PurchaseView{}, expected)

			jpy := "JPY"
			_, err := newUsecase(t, purchase, xr).CreatePurchase(
				context.Background(), CreatePurchaseParams{UserID: userID, DisplayCurrency: &jpy},
			)
			require.ErrorIs(t, err, expected)
		})
	})
}
