package handlertest

import (
	"context"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// NewBindHandlerTestInstance は、BindHandler用インスタンスを生成します。
func NewBindHandlerTestInstance(t *testing.T) (*echo.Echo, *gomock.Controller) {
	t.Helper()
	e := echo.New()
	ctrl := gomock.NewController(t)

	return e, ctrl
}

// NewImplementHandlerTestInstances は、エンドポイントのテスト用インスタンスを生成します。
func NewImplementHandlerTestInstances(t *testing.T) (
	context.Context, *gomock.Controller, *time.Location,
) {
	t.Helper()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	return ctx, ctrl, location
}
