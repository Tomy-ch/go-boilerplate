package version

import (
	"context"
	"net/http"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/testkit/testassert"
	"boilerplate-go/internal/controller/handler/version/gen"
	"boilerplate-go/internal/observability"
	"boilerplate-go/internal/system"
	mock_system "boilerplate-go/internal/system/mock"
	"boilerplate-go/pkg/datetime"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/version"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	tf := observability.NewNoopTracerFactory(t)

	bi := system.NewBuildInfo()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	loc, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	BindHandler(e, tf, loc, bi, appCfg)

	expectedMethods := []string{http.MethodGet}

	testassert.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	testassert.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func TestGetVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	lt := observability.NewMockControllerLayerTracer(t)

	bi := system.NewBuildInfo()
	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	osCfg := config.NewOperationSystemConfig(cfg)

	loc, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		s := &server{
			tracer:    lt,
			buildInfo: bi,
			appCfg:    appCfg,
			loc:       loc,
		}

		buildDate, err := datetime.ParseRFC3339UTCInLocation(bi.BuildDate(), loc)
		require.NoError(t, err)

		expectedResponse := gen.ResponseVersion{
			Version:     bi.Version(),
			Revision:    bi.Revision(),
			BuildDate:   buildDate,
			Environment: appCfg.Env(),
			Service:     appCfg.Name(),
		}

		resp, err := s.GetVersion(ctx, gen.GetVersionRequestObject{})
		require.NoError(t, err)

		actual, ok := resp.(gen.GetVersion200JSONResponse)
		require.True(t, ok)

		require.Equal(t, expectedResponse, gen.ResponseVersion(actual))
	})

	t.Run("異常系: ビルド日時のパースに失敗", func(t *testing.T) {
		t.Parallel()

		mockBuildInfo := mock_system.NewMockBuildInfo(ctrl)
		mockBuildInfo.EXPECT().BuildDate().Return("invalid-date-string").AnyTimes()
		s := &server{
			tracer:    lt,
			buildInfo: mockBuildInfo,
			appCfg:    appCfg,
			loc:       loc,
		}

		actual, err := s.GetVersion(ctx, gen.GetVersionRequestObject{})
		require.Nil(t, actual)
		require.Error(t, err)
	})
}
