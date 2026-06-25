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

func TestUsecaseConvert(t *testing.T) {
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
			got, err := uc.Convert(context.Background(), "USD", "JPY", 2)

			require.NoError(t, err)
			assert.InEpsilon(t, 300.0, got, 1e-9)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("gatewayがエラーを返したら換算せずそのまま伝播する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mock_exchangerate.NewMockGateway(ctrl)
			gw.EXPECT().
				GetRate(gomock.Any(), "USD", "JPY").
				Return(nil, xerrors.Wrap(apperror.ErrUnavailable, "downstream down"))

			uc := exchangerate.New(gw, observability.NewNoopTracerFactory(t))
			got, err := uc.Convert(context.Background(), "USD", "JPY", 2)

			require.ErrorIs(t, err, apperror.ErrUnavailable)
			assert.Zero(t, got)
		})
	})
}
