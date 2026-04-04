package hook

import (
	"context"
	"testing"

	mock_lifecycle "go-boilerplate/internal/di/lifecycle/mock"
	mock_driver "go-boilerplate/internal/infrastructure/rdb/driver/mock"
	mock_logging "go-boilerplate/internal/logging/mock"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegisterDBCloseHooks(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	reg := mock_lifecycle.NewMockRegistrar(ctrl)
	db := mock_driver.NewMockDatabaseDriver(ctrl)
	logger := mock_logging.NewMockLogger(ctrl)

	var closeFn func(context.Context) error
	dummy := func(context.Context) error { return nil }

	reg.EXPECT().RegisterStop(gomock.AssignableToTypeOf(dummy)).Do(func(args ...any) {
		closeFn = args[0].(func(context.Context) error)
	}).Times(1)

	RegisterDBCloseHooks(reg, db, logger)
	require.NotNil(t, closeFn)

	namedMock := mock_logging.NewMockLogger(ctrl)
	logger.EXPECT().Named("db.CloseHook").Return(namedMock)
	namedMock.EXPECT().Info("Closing database connection")
	db.EXPECT().Close()

	require.NoError(t, closeFn(context.Background()))
}
