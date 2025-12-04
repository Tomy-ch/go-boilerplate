package version

import (
	"net/http"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/handlertest/testassert"
	"boilerplate-go/internal/controller/handler/handlertest/testinstance"
	"boilerplate-go/internal/controller/handler/version/gen"
	"boilerplate-go/internal/system"
	"boilerplate-go/pkg/datetime"

	"github.com/stretchr/testify/require"
)

const targetPath = "/version"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e, _, tf, _ := testinstance.NewTestInstanceForBindHandler(t)
	bi := system.NewBuildInfo()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	osCfg := config.NewOSConfig(cfg)

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

	ctx, _, _, lt := testinstance.NewTestInstancesForImplementedUsecase(t)
	bi := system.NewBuildInfo()
	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	osCfg := config.NewOSConfig(cfg)

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

		invalidBI := system.NewTestBuildInfo(t, "1.0.0", "abc123", "invalid-date")
		s := &server{
			tracer:    lt,
			buildInfo: invalidBI,
			appCfg:    appCfg,
			loc:       loc,
		}

		actual, err := s.GetVersion(ctx, gen.GetVersionRequestObject{})
		require.Nil(t, actual)
		require.Error(t, err)
	})
}
