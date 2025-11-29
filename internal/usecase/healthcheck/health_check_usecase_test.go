package healthcheckuc

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/usecase/healthcheck/query"
	mock_query "boilerplate-go/internal/usecase/healthcheck/query/mock"
	"boilerplate-go/pkg/xerrors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNew(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	sysQuery := mock_query.NewMockDBSystemQuery(ctrl)

	cfg := &config.Config{}

	expected := &usecase{
		cfg:           cfg,
		dbSystemQuery: sysQuery,
	}
	actual := New(sysQuery, cfg)

	require.Equal(t, expected, actual)
}

func Test_usecase_CheckHealth(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.MockConfigForTest(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが正常な場合、OKステータスが返る", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			mockSysQuery := mock_query.NewMockDBSystemQuery(ctrl)

			mockSysQuery.EXPECT().CheckDBHealth(ctx).Return(query.DBHealth{
				Ready:       true,
				Latency:     1000,
				ResponsedAt: time.Now(),
			}, nil).Times(1)

			u := &usecase{
				dbSystemQuery: mockSysQuery,
				cfg:           cfg,
			}

			result, err := u.CheckHealth(ctx)
			require.NoError(t, err)
			require.Equal(t, Ok, result.Status)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが異常な場合、Unhealthyステータスが返る", func(t *testing.T) {
			t.Parallel()
			expectedDBHealth := query.DBHealth{
				Ready:       false,
				Latency:     0,
				ResponsedAt: time.Time{},
			}
			expectedResult := DTO{
				Status:          Unhealthy,
				ApplicationTime: time.Now(),
				DBHealthCheck:   expectedDBHealth,
			}
			expectedErr := xerrors.New("DB connection failed")

			ctrl := gomock.NewController(t)

			mockSysQuery := mock_query.NewMockDBSystemQuery(ctrl)
			mockSysQuery.EXPECT().CheckDBHealth(ctx).Return(expectedDBHealth, expectedErr).Times(1)

			u := &usecase{
				cfg:           cfg,
				dbSystemQuery: mockSysQuery,
			}

			actualResult, actualErr := u.CheckHealth(ctx)
			require.Equal(t, expectedErr, actualErr)
			require.Equal(t, expectedResult.Status, actualResult.Status)
			require.WithinDuration(t, expectedResult.ApplicationTime, actualResult.ApplicationTime, time.Second)
			require.Equal(t, expectedResult.DBHealthCheck, actualResult.DBHealthCheck)
		})
	})
}
