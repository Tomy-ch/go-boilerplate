package shutdowner

import (
	"errors"
	"testing"

	mock_shutdowner "go-boilerplate/internal/di/shutdowner/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestNewShutdowner(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx.Shutdownerを渡すと非nilなShutdownerを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fxSD := mock_shutdowner.NewMockFxShutdowner(ctrl)

			assert.NotNil(t, NewShutdowner(fxSD))
		})
	})
}

func Test_Shutdowner_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx.ShutdownerのShutdownを呼び出しnilを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			fxSD := mock_shutdowner.NewMockFxShutdowner(ctrl)
			fxSD.EXPECT().Shutdown().Return(nil)

			err := NewShutdowner(fxSD).Shutdown()
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("fx.ShutdownerのShutdownのエラーをそのまま返す", func(t *testing.T) {
			t.Parallel()

			expectedErr := errors.New("shutdown error")
			ctrl := gomock.NewController(t)
			fxSD := mock_shutdowner.NewMockFxShutdowner(ctrl)
			fxSD.EXPECT().Shutdown().Return(expectedErr)

			err := NewShutdowner(fxSD).Shutdown()
			require.ErrorIs(t, err, expectedErr)
		})
	})
}
