package hook

import (
	"context"
	"errors"
	"testing"

	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	mock_logging "go-boilerplate/internal/logging/mock"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterDBCloseHooks(t *testing.T) {
	t.Parallel()

	// captureCloseFn は、RegisterDBCloseHooks が RegisterStop に登録する close 関数を捕捉して返す。
	captureCloseFn := func(t *testing.T, ctrl *gomock.Controller, db *mock_driver.MockDatabaseDriver, logger *mock_logging.MockLogger) func(context.Context) error {
		t.Helper()

		reg := mock_lifecycle.NewMockRegistrar(ctrl)
		var closeFn func(context.Context) error
		dummy := func(context.Context) error { return nil }
		reg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
			fn, ok := args[0].(func(context.Context) error)
			require.True(t, ok)
			closeFn = fn
		}).Times(1)

		RegisterDBCloseHooks(reg, db, logger)
		require.NotNil(t, closeFn)
		return closeFn
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("OnStopでNamedロガーにInfoを出しDBを閉じる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			logger := mock_logging.NewMockLogger(ctrl)
			closeFn := captureCloseFn(t, ctrl, db, logger)

			namedMock := mock_logging.NewMockLogger(ctrl)
			logger.EXPECT().Named("db.CloseHook").Return(namedMock)
			namedMock.EXPECT().Info("Closing database connection")
			db.EXPECT().Close()

			require.NoError(t, closeFn(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのClose失敗時はErrorログを出しエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			db := mock_driver.NewMockDatabaseDriver(ctrl)
			logger := mock_logging.NewMockLogger(ctrl)
			closeFn := captureCloseFn(t, ctrl, db, logger)

			namedMock := mock_logging.NewMockLogger(ctrl)
			logger.EXPECT().Named("db.CloseHook").Return(namedMock)
			namedMock.EXPECT().Info("Closing database connection")
			wantErr := errors.New("close failed")
			db.EXPECT().Close().Return(wantErr)
			namedMock.EXPECT().Error("failed to close database", gomock.Any()).Times(1)

			require.ErrorIs(t, closeFn(context.Background()), wantErr)
		})
	})
}
