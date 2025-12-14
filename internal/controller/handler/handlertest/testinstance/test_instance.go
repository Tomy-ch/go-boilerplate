// Package testinstance は、ハンドラー用のテストインスタンス生成ユーティリティを提供します。
package testinstance

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/logging"
	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// NewTestInstanceForBindHandler は、BindHandler用インスタンスを生成します。
func NewTestInstanceForBindHandler(t *testing.T) (
	*echo.Echo, *gomock.Controller, observability.TracerFactory, logging.Logger,
) {
	t.Helper()
	e := echo.New()
	ctrl := gomock.NewController(t)

	log := logging.NewTestLogger(t)
	tf := observability.NewNoopTracerFactory(t)
	return e, ctrl, tf, log
}

// NewTestInstancesForImplementedUsecase は、エンドポイントのテスト用インスタンスを生成します。
func NewTestInstancesForImplementedUsecase(t *testing.T) (
	context.Context, *gomock.Controller, *time.Location, observability.LayerTracer,
) {
	t.Helper()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	tf := observability.NewNoopTracerFactory(t)
	lt := tf.Controller()

	return ctx, ctrl, location, lt
}
