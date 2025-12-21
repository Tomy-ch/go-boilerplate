package ready

import (
	"context"
	"net/http"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/controller/handler/handlertest/testassert"
	"boilerplate-go/internal/controller/handler/ready/gen"
	"boilerplate-go/internal/observability"
	healthcheckuc "boilerplate-go/internal/usecase/healthcheck"
	mock_healthcheckuc "boilerplate-go/internal/usecase/healthcheck/mock"
	"boilerplate-go/internal/usecase/healthcheck/query"
	"boilerplate-go/pkg/xerrors"

	"github.com/labstack/echo/v4"
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

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	lt := observability.NewMockControllerLayerTracer(t)
	loc := config.NewTestLocation(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("レディネスチェックが成功する", func(t *testing.T) {
			t.Parallel()
			expectedStatus := healthcheckuc.Ok
			expectedAppTime := time.Date(2024, time.June, 1, 12, 0, 1, 0, loc)
			expectedDBResAt := time.Date(2024, time.June, 1, 12, 0, 2, 0, loc)
			expectedDBLatency := time.Duration(1500000)

			uc := mock_healthcheckuc.NewMockUsecase(ctrl)
			uc.EXPECT().CheckHealth(gomock.Any()).Return(
				healthcheckuc.DTO{
					Status:          expectedStatus,
					ApplicationTime: expectedAppTime,
					DBHealthCheck: query.DBHealth{
						Latency:     expectedDBLatency,
						ResponsedAt: expectedDBResAt,
					},
				}, nil)

			s := &server{
				tracer:        lt,
				healthUsecase: uc,
			}

			expected := gen.GetReady200JSONResponse(gen.ResponseReady{
				Status:          gen.ResponseReadyStatus(expectedStatus),
				ApplicationTime: expectedAppTime,
				DbLatencyMs:     expectedDBLatency.Milliseconds(),
				DbResponsedAt:   expectedDBResAt,
			})

			resp, err := s.GetReady(ctx, gen.GetReadyRequestObject{})
			require.NoError(t, err)

			actual, ok := resp.(gen.GetReady200JSONResponse)
			require.True(t, ok)

			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのレディネスチェックが失敗する", func(t *testing.T) {
			t.Parallel()
			expectedErr := xerrors.New("example error")

			uc := mock_healthcheckuc.NewMockUsecase(ctrl)
			uc.EXPECT().CheckHealth(gomock.Any()).Return(
				healthcheckuc.DTO{},
				expectedErr,
			)
			s := &server{
				tracer:        lt,
				healthUsecase: uc,
			}

			res, actualErr := s.GetReady(ctx, gen.GetReadyRequestObject{})
			require.Nil(t, res)
			require.ErrorIs(t, expectedErr, actualErr)
		})
	})
}
