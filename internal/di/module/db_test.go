package module

import (
	"testing"

	"github.com/exaring/otelpgx"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	gomock "go.uber.org/mock/gomock"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/di/lifecycle"
	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/rdb/metrics"
	"go-boilerplate/internal/logging"
	mock_logging "go-boilerplate/internal/logging/mock"
	"go-boilerplate/internal/observability"
	"go-boilerplate/internal/usecase/boundary/tx"
)

func TestDatabaseModule_Composes(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DatabaseModule を含む fx アプリの依存グラフが解決できる", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockLogger := mock_logging.NewMockLogger(ctrl)
			mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)

			// fx.ValidateApp は依存グラフの解決可能性のみを検証し、コンストラクタや
			// ライフサイクルフックを実行しない。NewTracedDB 内の実 DB 接続(Ping)が
			// 走らないため、Postgres 不在でも DatabaseModule の合成を検証できる。
			require.NoError(t, fx.ValidateApp(
				lifecycle.Module(),
				DatabaseModule(),
				fx.Provide(func() testing.TB { return t }),
				fx.Provide(config.MockConfigForTest),
				fx.Provide(config.NewDatabaseConfig),
				fx.Provide(config.NewObservabilityConfig),
				fx.Provide(config.NewOperatingSystemConfig),
				fx.Provide(config.NewDBConnectionConfig),
				fx.Provide(func() logging.Logger { return mockLogger }),
				fx.Provide(func() logging.LogFieldBuilder { return mockLF }),
				fx.Provide(func() *otelpgx.Tracer { return otelpgx.NewTracer() }),
				fx.Provide(observability.NewNoopTracerFactory),
				fx.NopLogger,
			))
		})
	})
}

func TestDatabaseModule(t *testing.T) {
	t.Parallel()

	// DatabaseModule は config / logging / pgx tracer を外から受け取る。
	dbDeps := func(t *testing.T) []fx.Option {
		t.Helper()

		ctrl := gomock.NewController(t)
		mockLogger := mock_logging.NewMockLogger(ctrl)
		mockLF := mock_logging.NewMockLogFieldBuilder(ctrl)

		return []fx.Option{
			lifecycle.Module(),
			fx.Provide(func() testing.TB { return t }),
			fx.Provide(config.MockConfigForTest),
			fx.Provide(config.NewDatabaseConfig),
			fx.Provide(config.NewObservabilityConfig),
			fx.Provide(config.NewOperatingSystemConfig),
			fx.Provide(config.NewDBConnectionConfig),
			fx.Provide(func() logging.Logger { return mockLogger }),
			fx.Provide(func() logging.LogFieldBuilder { return mockLF }),
			fx.Provide(func() *otelpgx.Tracer { return otelpgx.NewTracer() }),
			fx.Provide(observability.NewNoopTracerFactory),
			fx.NopLogger,
		}
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DB 接続とトランザクションマネージャとプール計測を提供する", func(t *testing.T) {
			t.Parallel()

			var (
				db        driver.DatabaseDriver
				txm       tx.Manager
				collector *metrics.PoolStatsCollector
			)

			// tx.Manager は clock.Sleeper を要求するため、clock substrate も併せて配線する。
			opts := append(dbDeps(t), clockModule(), DatabaseModule(), fx.Populate(&db, &txm, &collector))
			require.NoError(t, fx.ValidateApp(opts...))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("未配線では DB 接続が解決できずグラフ検証に失敗する", func(t *testing.T) {
			t.Parallel()

			var db driver.DatabaseDriver

			opts := append(dbDeps(t), fx.Populate(&db))
			require.Error(t, fx.ValidateApp(opts...))
		})
	})
}
