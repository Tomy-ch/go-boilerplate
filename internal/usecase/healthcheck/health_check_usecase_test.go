package healthcheck

import (
	"context"
	"testing"
	"time"

	"go-boilerplate/internal/observability"
	clocktest "go-boilerplate/internal/usecase/boundary/clock/testkit"
	"go-boilerplate/internal/usecase/healthcheck/query"
	mock_query "go-boilerplate/internal/usecase/healthcheck/query/mock"
	"go-boilerplate/internal/usecase/testkit"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// errProbe は、依存の検査が失敗したことを表すテスト用のエラーです。
var errProbe = xerrors.New("probe failed")

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		tf := observability.NewNoopTracerFactory(t)
		sysQuery := mock_query.NewMockDBSystemCqrs(ctrl)
		clock := clocktest.NewMockClock(t, time.Time{})

		probes := []Probe{{Name: "dep"}}
		expected := &usecase{
			tracer:       tf.Usecase(),
			clock:        clock,
			dbSystemCqrs: sysQuery,
			probes:       probes,
		}
		actual := New(sysQuery, tf, clock, probes)

		assert.Equal(t, expected, actual)
	})
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

			mockSysQuery := mock_query.NewMockDBSystemCqrs(ctrl)
			mockSysQuery.EXPECT().CheckDBHealth(gomock.Any()).Return(query.DBHealth{
				Ready:       true,
				Latency:     1000,
				RespondedAt: now,
			}, nil).Times(1)

			mockClock := clocktest.NewMockClockOnce(t, now)

			u := &usecase{
				tracer:       lt,
				clock:        mockClock,
				dbSystemCqrs: mockSysQuery,
			}

			result, err := u.CheckHealth(ctx)
			require.NoError(t, err)
			assert.Equal(t, Ok, result.Status)
			assert.Equal(t, now, result.ApplicationTime)
			assert.Empty(t, result.Dependencies, "probe が無ければ依存は並ばないこと")
		})

		t.Run("probe がすべて通れば ok のまま依存の状態が並ぶ", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			mockSysQuery := mock_query.NewMockDBSystemCqrs(ctrl)
			mockSysQuery.EXPECT().CheckDBHealth(gomock.Any()).Return(query.DBHealth{Ready: true}, nil).Times(1)

			u := &usecase{
				tracer:       observability.NewMockUsecaseLayerTracer(t),
				clock:        clocktest.NewMockClockOnce(t, now),
				dbSystemCqrs: mockSysQuery,
				probes: []Probe{
					{Name: "realtime", Check: func(context.Context) error { return nil }},
				},
			}

			result, err := u.CheckHealth(ctx)

			require.NoError(t, err)
			assert.Equal(t, Ok, result.Status)
			assert.Equal(t, []DependencyStatus{{Name: "realtime", Status: Ok}}, result.Dependencies)
		})

		t.Run("probe が落ちても DB が健全なら degraded を返しエラーにしない", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			mockSysQuery := mock_query.NewMockDBSystemCqrs(ctrl)
			mockSysQuery.EXPECT().CheckDBHealth(gomock.Any()).Return(query.DBHealth{Ready: true}, nil).Times(1)

			u := &usecase{
				tracer:       observability.NewMockUsecaseLayerTracer(t),
				clock:        clocktest.NewMockClockOnce(t, now),
				dbSystemCqrs: mockSysQuery,
				probes: []Probe{
					{Name: "healthy", Check: func(context.Context) error { return nil }},
					{Name: "realtime", Check: func(context.Context) error { return errProbe }},
				},
			}

			result, err := u.CheckHealth(ctx)

			// エラーにしないことが要点。ここで 503 を返すと Realtime だけの不調で instance が
			// load balancer から外れ、通常の HTTP まで止まる。
			require.NoError(t, err)
			assert.Equal(t, Degraded, result.Status)
			assert.Equal(t, []DependencyStatus{
				{Name: "healthy", Status: Ok},
				{Name: "realtime", Status: Degraded},
			}, result.Dependencies)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DBのヘルスチェックが異常な場合、空のDTOとエラーが返る", func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			lt := observability.NewMockUsecaseLayerTracer(t)
			now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

			expectedErr := testkit.ExpectedDBError()

			mockSysQuery := mock_query.NewMockDBSystemCqrs(ctrl)
			mockSysQuery.EXPECT().CheckDBHealth(gomock.Any()).Return(query.DBHealth{}, expectedErr).Times(1)

			mockClock := clocktest.NewMockClockOnce(t, now)

			u := &usecase{
				tracer:       lt,
				clock:        mockClock,
				dbSystemCqrs: mockSysQuery,
			}

			actualResult, actualErr := u.CheckHealth(ctx)
			require.ErrorIs(t, actualErr, expectedErr)
			assert.Nil(t, actualResult)
		})
	})
}
