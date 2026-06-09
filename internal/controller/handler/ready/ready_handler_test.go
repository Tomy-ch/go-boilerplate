package ready

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/controller/handler/ready/gen"
	"go-boilerplate/internal/controller/handler/testkit/testassert"
	"go-boilerplate/internal/observability"
	healthcheckuc "go-boilerplate/internal/usecase/healthcheck"
	mock_healthcheckuc "go-boilerplate/internal/usecase/healthcheck/mock"
	"go-boilerplate/internal/usecase/healthcheck/query"
	"go-boilerplate/pkg/xerrors"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const targetPath = "/ready"

func TestBindHandler(t *testing.T) {
	t.Parallel()

	e := echo.New()
	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)

	uc := mock_healthcheckuc.NewMockUsecase(ctrl)

	BindHandler(e, tf, uc)

	expectedMethods := []string{http.MethodGet}

	testassert.AssertEchoRouterPath(
		t, targetPath, e.Routes(),
	)
	testassert.AssertEchoRouterMethods(
		t, expectedMethods, e.Routes(),
	)
}

func TestGetReady(t *testing.T) {
	t.Parallel()

	loc := config.NewTestLocation(t)

	t.Run("正常系_レディネスチェックが成功し200を返す", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctrl := gomock.NewController(t)

		appTime := time.Date(2024, time.June, 1, 12, 0, 1, 0, loc)
		dbResAt := time.Date(2024, time.June, 1, 12, 0, 2, 0, loc)

		uc := mock_healthcheckuc.NewMockUsecase(ctrl)
		uc.EXPECT().CheckHealth(gomock.Any()).Return(
			healthcheckuc.DTO{
				Status:          healthcheckuc.Ok,
				ApplicationTime: appTime,
				DBHealthCheck: query.DBHealth{
					Latency:     1500 * time.Microsecond,
					ResponsedAt: dbResAt,
				},
			}, nil)

		s := &server{
			tracer:        observability.NewMockControllerLayerTracer(t),
			healthUsecase: uc,
		}

		resp, err := s.GetReady(ctx, gen.GetReadyRequestObject{})
		require.NoError(t, err)

		actual, ok := resp.(gen.GetReady200JSONResponse)
		require.True(t, ok)

		expected := gen.GetReady200JSONResponse(gen.ReadyResponse{
			Status:          gen.Ok,
			ApplicationTime: appTime,
			DbLatencyMs:     1,
			DbResponsedAt:   dbResAt,
		})
		assert.Equal(t, expected, actual)
	})

	t.Run("異常系_DBのレディネスチェックが失敗するとエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctrl := gomock.NewController(t)
		expectedErr := xerrors.New("example error")

		uc := mock_healthcheckuc.NewMockUsecase(ctrl)
		uc.EXPECT().CheckHealth(gomock.Any()).Return(
			healthcheckuc.DTO{},
			expectedErr,
		)
		s := &server{
			tracer:        observability.NewMockControllerLayerTracer(t),
			healthUsecase: uc,
		}

		res, err := s.GetReady(ctx, gen.GetReadyRequestObject{})
		require.Nil(t, res)
		require.ErrorIs(t, err, expectedErr)
	})
}
