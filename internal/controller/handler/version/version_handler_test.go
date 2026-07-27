package version

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/controller/handler/version/gen"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/system"
	mock_system "go-boilerplate/internal/system/mock"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
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
	osCfg := config.NewOperatingSystemConfig(cfg)

	loc, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	BindHandler(e, tf, loc, bi, appCfg)

	expectedMethods := []string{http.MethodGet}

	testassert.AssertEchoRouterPath(
		t, targetPath, e.Router().Routes(),
	)
	testassert.AssertEchoRouterMethods(
		t, expectedMethods, e.Router().Routes(),
	)
}

func Test_server_GetVersion(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	appCfg := config.NewApplicationConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)

	loc, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("バージョン情報を返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)

			bi := mock_system.NewMockBuildInfo(ctrl)
			bi.EXPECT().Version().Return("v1.2.3")
			bi.EXPECT().Revision().Return("abc1234")
			bi.EXPECT().BuildDate().Return("2024-01-02T03:04:05Z")

			s := &server{
				tracer:    observability.NewMockControllerLayerTracer(t),
				buildInfo: bi,
				appCfg:    appCfg,
				loc:       loc,
			}

			resp, err := s.GetVersion(ctx, gen.GetVersionRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetVersion200JSONResponse)
			require.True(t, ok)

			expectedResponse := gen.VersionResponse{
				Version:     "v1.2.3",
				Revision:    "abc1234",
				BuildDate:   time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC).In(loc),
				Environment: appCfg.Env(),
				Service:     appCfg.Name(),
			}
			assert.Equal(t, expectedResponse, gen.VersionResponse(actual))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ビルド日時のパースに失敗するとErrInternalを返す", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctrl := gomock.NewController(t)

			mockBuildInfo := mock_system.NewMockBuildInfo(ctrl)
			mockBuildInfo.EXPECT().BuildDate().Return("invalid-date-string")
			s := &server{
				tracer:    observability.NewMockControllerLayerTracer(t),
				buildInfo: mockBuildInfo,
				appCfg:    appCfg,
				loc:       loc,
			}

			actual, err := s.GetVersion(ctx, gen.GetVersionRequestObject{})
			require.Nil(t, actual)
			require.ErrorIs(t, err, errInvalidBuildDate)
		})
	})
}
