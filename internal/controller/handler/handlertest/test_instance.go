package handlertest

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/observability"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// NewTestInstanceForBindHandler は、BindHandler用インスタンスを生成します。
func NewTestInstanceForBindHandler(t *testing.T) (
	*echo.Echo, *gomock.Controller, observability.TracerFactory, *zap.Logger,
) {
	t.Helper()
	e := echo.New()
	ctrl := gomock.NewController(t)

	tp := noop.NewTracerProvider()
	z := zap.NewNop()
	tf := observability.NewTracerFactory(tp, z)

	return e, ctrl, tf, z
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

	tp := noop.NewTracerProvider()
	z := zap.NewNop()
	tf := observability.NewTracerFactory(tp, z)
	lt := tf.Controller()

	return ctx, ctrl, location, lt
}
