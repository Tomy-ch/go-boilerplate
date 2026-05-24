package healthcheck

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/observability"
	mock_clock "go-boilerplate/internal/usecase/boundary/clock/mock"
	"go-boilerplate/internal/usecase/healthcheck/query"
	mock_query "go-boilerplate/internal/usecase/healthcheck/query/mock"
	"go-boilerplate/internal/usecase/testkit"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	tf := observability.NewNoopTracerFactory(t)
	sysQuery := mock_query.NewMockDBSystemQuery(ctrl)
	clock := mock_clock.NewMockClock(ctrl)

	expected := &usecase{
		tracer:        tf.Usecase(),
		clock:         clock,
		dbSystemQuery: sysQuery,
	}
	actual := New(sysQuery, tf, clock)

	require.Equal(t, expected, actual)
}

func Test_usecase_CheckHealth(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが正常な場合、OKステータスが返る", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			mockSysQuery := mock_query.NewMockDBSystemQuery(ctrl)
			mockSysQuery.EXPECT().CheckDBHealth(gomock.Any()).Return(query.DBHealth{
				Ready:       true,
				Latency:     1000,
				ResponsedAt: now,
			}, nil).Times(1)

			mockClock := mock_clock.NewMockClock(ctrl)
			mockClock.EXPECT().Now().Return(now).Times(1)

			u := &usecase{
				tracer:        lt,
				clock:         mockClock,
				dbSystemQuery: mockSysQuery,
			}

			result, err := u.CheckHealth(ctx)
			require.NoError(t, err)
			require.Equal(t, Ok, result.Status)
			require.Equal(t, now, result.ApplicationTime)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが異常な場合、Unhealthyステータスが返る", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			expectedDBHealth := query.DBHealth{
				Ready:       false,
				Latency:     0,
				ResponsedAt: time.Time{},
			}
			expectedResult := DTO{
				Status:          Unhealthy,
				ApplicationTime: now,
				DBHealthCheck:   expectedDBHealth,
			}
			expectedErr := testkit.ExpectedDBError(t)

			mockSysQuery := mock_query.NewMockDBSystemQuery(ctrl)
			mockSysQuery.EXPECT().CheckDBHealth(gomock.Any()).Return(expectedDBHealth, expectedErr).Times(1)

			mockClock := mock_clock.NewMockClock(ctrl)
			mockClock.EXPECT().Now().Return(now).Times(1)

			u := &usecase{
				tracer:        lt,
				clock:         mockClock,
				dbSystemQuery: mockSysQuery,
			}

			actualResult, actualErr := u.CheckHealth(ctx)
			require.Equal(t, expectedErr, actualErr)
			require.Equal(t, expectedResult.Status, actualResult.Status)
			require.Equal(t, expectedResult.ApplicationTime, actualResult.ApplicationTime)
			require.Equal(t, expectedResult.DBHealthCheck, actualResult.DBHealthCheck)
		})
	})
}
