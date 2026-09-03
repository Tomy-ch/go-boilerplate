package module

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/di/shutdowner"
	"go-boilerplate/internal/logging"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/job"
)

// realtimeCleanupRunDeps は、constructor を実際に走らせて jobs グループを集めるための依存です。
// ConfigModule / ObservabilityModule / DatabaseModule は global（設定の読み込みと prometheus の既定
// registry）へ触るため並列テストと競合します。cleanup module が要るのは設定 2 つと
// Logger / TracerFactory / outbound クライアントだけなので、いずれも test 用の実装を直接差し込みます。
//
// fan-out を差し替えていないことが要点です。REALTIME_TOPIC が空の設定のままジョブを登録できることを
// 確かめるために、意図して素のまま組み立てます。
func realtimeCleanupRunDeps(t *testing.T) fx.Option {
	t.Helper()

	cfg := config.MockConfigForTest(t)

	return fx.Options(
		lifecycle.Module(), clockModule(), RealtimeCleanupModule(),
		fx.Provide(func() *config.EndpointConfig { return config.NewEndpointConfig(cfg) }),
		fx.Provide(func() *config.RealtimeConfig { return config.NewRealtimeConfig(cfg) }),
		fx.Provide(func() logging.Logger { return logging.NewTestLogger(t) }),
		fx.Provide(func() observability.TracerFactory { return observability.NewNoopTracerFactory(t) }),
		fx.Provide(func() *observability.OutboundHTTPClient { return observability.NewDisabledOutboundHTTPClient(true) }),
		fx.Replace(observability.NewNoopRealtimeMetrics(t)),
	)
}

func TestRealtimeCleanupModule_GraphIsValid(t *testing.T) {
	t.Parallel()

	// 実際の配線と同じ組み合わせ（internal/di/job.go の NewJobCore）で検証する。
	// realtimeModule() とは同じ graph に載らないので、ここでも同居させない。
	validateGraph(t, append(commonDeps(),
		shutdowner.Module(),
		InfrastructureModule(), UsecaseModule(),
		JobModule(), RealtimeCleanupModule(),
	)...)
}

func TestRealtimeCleanupModule(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Realtime が未設定でも jobs グループを組める", func(t *testing.T) {
			t.Parallel()

			// これが崩れると、REALTIME_TOPIC が空の環境（dast / dev / stg / prd）で outbox-gc のような
			// 無関係なジョブまで起動できなくなる。fx は Runner を組むために登録済みジョブの constructor を
			// すべて実行するので、Realtime の構築が graph に載っていないことをここで締める。
			require.Empty(t, config.NewRealtimeConfig(config.MockConfigForTest(t)).Topic())

			got := collectGroup[job.Job](t, `group:"jobs"`, realtimeCleanupRunDeps(t))

			names := make([]string, 0, len(got))
			for _, j := range got {
				names = append(names, j.Name())
			}
			assert.Equal(t, []string{"orphan-cleanup"}, names)
		})
	})
}

func Test_provideOrphanSweeperFactory(t *testing.T) {
	t.Parallel()

	cfg := config.MockConfigForTest(t)
	factory := provideOrphanSweeperFactory(
		config.NewRealtimeConfig(cfg),
		config.NewEndpointConfig(cfg),
		observability.NewDisabledOutboundHTTPClient(true),
		nil,
		observability.NewNoopTracerFactory(t),
	)
	require.NotNil(t, factory)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("topic が設定されていれば掃除役を組み立てる", func(t *testing.T) {
			t.Parallel()

			// クライアント構築は接続しないので、emulator が居なくてもここまで通る。
			// endpoint の振り分け（store 側と fan-out 側）や table 名の配線ミスを、job 実行前に捕まえる。
			cfg := config.MockConfigForTest(t)
			rtCfg := config.NewRealtimeConfig(cfg)
			rtCfg.SetTopic(t, "arn:aws:sns:us-east-1:000000000000:realtime")

			sweeper, err := provideOrphanSweeperFactory(
				rtCfg,
				config.NewEndpointConfig(cfg),
				observability.NewDisabledOutboundHTTPClient(true),
				nil,
				observability.NewNoopTracerFactory(t),
			)(t.Context())
			require.NoError(t, err)
			assert.NotNil(t, sweeper)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Realtime が未設定なら、掃除役を組む時点で初めて失敗する", func(t *testing.T) {
			t.Parallel()

			sweeper, err := factory(t.Context())
			require.ErrorIs(t, err, ErrRealtimeTopicNotConfigured)
			assert.Nil(t, sweeper)
		})
	})
}
