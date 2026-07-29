package exchangerate_test

import (
	"context"
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	boundary "go-boilerplate/internal/usecase/boundary/exchangerate"
	mock_exchangerate "go-boilerplate/internal/usecase/boundary/exchangerate/mock"
	"go-boilerplate/internal/usecase/exchangerate"
	"go-boilerplate/pkg/decimal"
	decimaltestkit "go-boilerplate/pkg/decimal/testkit"
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

			got := exchangerate.New(gw, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))

			assert.NotNil(t, got)
		})
	})
}

func TestBuildReferenceAmount(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("レートと金額から参考換算額を最小単位整数で組み立てる", func(t *testing.T) {
			t.Parallel()
			rate := &boundary.Rate{Base: "USD", Quote: "JPY", Value: decimaltestkit.MustParse(t, "150.5"), Date: "2026-07-21"}
			ref, err := exchangerate.BuildReferenceAmount(rate, decimal.FromInt(100))
			require.NoError(t, err)
			require.NotNil(t, ref)
			assert.Equal(t, int64(15050), ref.Amount)
			assert.Equal(t, "JPY", ref.Currency)
			assert.Equal(t, "150.5", ref.Rate.String())
			assert.Equal(t, "2026-07-21", ref.RateDate)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("最小単位整数が int64 の範囲を超える場合はエラーを返す", func(t *testing.T) {
			t.Parallel()
			rate := &boundary.Rate{Base: "USD", Quote: "JPY", Value: decimal.FromInt(1)}
			_, err := exchangerate.BuildReferenceAmount(rate, decimaltestkit.MustParse(t, "1e19"))
			require.ErrorIs(t, err, decimal.ErrOverflow)
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
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: decimal.FromInt(150)}, nil)

			uc := exchangerate.New(gw, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{Base: "USD", Quote: "JPY", Amount: decimal.FromInt(2)})

			require.NoError(t, err)
			assert.Equal(t, "300", got.Converted.String())
			assert.Nil(t, got.Reference)
		})

		t.Run("displayCurrency指定時は参考換算額を整数で返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			// 本体換算 base→quote（USD→EUR）
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "EUR").
				Return(&boundary.Rate{Base: "USD", Quote: "EUR", Value: decimaltestkit.MustParse(t, "0.92")}, nil)
			// 参考換算 base→display（USD→JPY）
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: decimaltestkit.MustParse(t, "150.5"), Date: "2026-07-21"}, nil)

			uc := exchangerate.New(gw, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{
				Base: "USD", Quote: "EUR", Amount: decimal.FromInt(100), DisplayCurrency: new("JPY"),
			})

			require.NoError(t, err)
			require.NotNil(t, got.Reference)
			// 100 USD × 150.5 = 15050 JPY（JPY は最小単位 0 桁）
			assert.Equal(t, int64(15050), got.Reference.Amount)
			assert.Equal(t, "JPY", got.Reference.Currency)
			assert.Equal(t, "2026-07-21", got.Reference.RateDate)
			assert.Equal(t, "150.5", got.Reference.Rate.String())
		})

		t.Run("degrade: 参考換算のレート取得失敗でも本体換算は継続しReferenceはnil", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			// 本体換算は成功
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "USD").
				Return(&boundary.Rate{Base: "USD", Quote: "USD", Value: decimal.FromInt(1)}, nil)
			// 参考換算 base→display は失敗
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "reference rate down"))

			uc := exchangerate.New(gw, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{
				Base: "USD", Quote: "USD", Amount: decimal.FromInt(100), DisplayCurrency: new("JPY"),
			})

			require.NoError(t, err)
			assert.Equal(t, "100", got.Converted.String())
			assert.Nil(t, got.Reference)
		})

		t.Run("degrade: 参考換算額が最小単位整数の範囲外でも本体換算は継続しReferenceはnil", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			// 本体換算・参考換算のレート取得はいずれも成功する
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "EUR").
				Return(&boundary.Rate{Base: "USD", Quote: "EUR", Value: decimaltestkit.MustParse(t, "0.92")}, nil)
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(&boundary.Rate{Base: "USD", Quote: "JPY", Value: decimal.FromInt(1)}, nil)

			uc := exchangerate.New(gw, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))
			// amount×rate が int64 を超え BuildReferenceAmount が overflow するため参考換算は degrade する
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{
				Base: "USD", Quote: "EUR", Amount: decimaltestkit.MustParse(t, "1e19"), DisplayCurrency: new("JPY"),
			})

			require.NoError(t, err)
			assert.Equal(t, "9200000000000000000", got.Converted.String())
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

			uc := exchangerate.New(gw, logging.NewTestLogger(t), observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), exchangerate.ConvertInput{Base: "USD", Quote: "JPY", Amount: decimal.FromInt(2)})

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Nil(t, got)
		})
	})
}
