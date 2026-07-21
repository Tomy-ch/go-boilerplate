package exchangerate_test

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
	mock_exchangerate "go-boilerplate/internal/usecase/boundary/exchangerate/mock"
	"go-boilerplate/internal/usecase/exchangerate"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("依存を渡すと非nilのUsecaseを生成する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)

			got := exchangerate.New(gw, observability.NewNoopTracerFactory(t))

			assert.NotNil(t, got)
		})
	})
}

func Test_usecase_Convert(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("gatewayのレートでamountを換算する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: 150}, nil)

			uc := exchangerate.New(gw, observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{Base: "USD", Quote: "JPY", Amount: 2})

			require.NoError(t, err)
			assert.InEpsilon(t, 300.0, got.Converted, 1e-9)
			assert.Nil(t, got.Reference)
		})

		t.Run("display_currency指定時は参考換算額を整数で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			// 本体換算 base→quote（USD→EUR）
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "EUR").
				Return(&boundary.Rate{Base: "USD", Quote: "EUR", Value: 0.92}, nil)
			// 参考換算 base→display（USD→JPY）
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: 150.5, Date: "2026-07-21"}, nil)

			uc := exchangerate.New(gw, observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{
				Base: "USD", Quote: "EUR", Amount: 100, DisplayCurrency: new("JPY"),
			})

			require.NoError(t, err)
			require.NotNil(t, got.Reference)
			// 100.00 USD (10000 cent) × 150.5 / 100 = 15050 JPY
			assert.Equal(t, int64(15050), got.Reference.Amount)
			assert.Equal(t, "JPY", got.Reference.Currency)
			assert.Equal(t, "2026-07-21", got.Reference.RateDate)
			assert.InEpsilon(t, 150.5, got.Reference.Rate, 1e-9)
		})

		t.Run("degrade: 参考換算のレート取得失敗でも本体換算は継続しReferenceはnil", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			// 本体換算は成功
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "USD").
				Return(&boundary.Rate{Base: "USD", Quote: "USD", Value: 1}, nil)
			// 参考換算 base→display は失敗
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "reference rate down"))

			uc := exchangerate.New(gw, observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{
				Base: "USD", Quote: "USD", Amount: 100, DisplayCurrency: new("JPY"),
			})

			require.NoError(t, err)
			assert.InEpsilon(t, 100.0, got.Converted, 1e-9)
			assert.Nil(t, got.Reference)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("本体換算のレート取得失敗はそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "downstream down"))

			uc := exchangerate.New(gw, observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{Base: "USD", Quote: "JPY", Amount: 2})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, got)
		})
	})
}
