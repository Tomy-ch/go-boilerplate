package module

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"boilerplate-go/internal/config"
)

func TestConfigConstructors_WithProvidedConfig(t *testing.T) {
	t.Run("ConfigModule を使って各コンポーネントが生成される", func(t *testing.T) {
		// テスト環境の .env を読み込むためリポジトリルートに移動し ENV を設定する
		config.EnsureRepoRootAndEnv(t, config.TestingEnvValue)

		cfg, err := config.SetUpConfig()
		require.NoError(t, err)

		var (
			osCfg      *config.OperationSystemConfig
			appCfg     *config.ApplicationConfig
			serverCfg  *config.ServerConfig
			metricsCfg *config.MetricsConfig
			obsCfg     *config.ObservabilityConfig
			dbCfg      *config.DatabaseConfig
			dbConnCfg  *config.DBConnectionConfig
			secCfg     *config.SecurityConfig
			ipCfg      *config.IPRateLimitConfig
			loc        *time.Location
		)

		app := fx.New(
			fx.Provide(func() testing.TB { return t }),
			// テスト対象: 実装側のモジュール
			ConfigModule(),
			fx.Populate(&osCfg, &appCfg, &serverCfg, &dbCfg, &dbConnCfg, &metricsCfg, &obsCfg, &secCfg, &ipCfg, &loc),
			fx.NopLogger,
		)

		require.NoError(t, app.Start(context.Background()))
		defer func() { require.NoError(t, app.Stop(context.Background())) }()

		// fx で注入された結果が SetUpConfig を通して得られる値と一致することを確認
		require.Equal(t, config.NewOperationSystemConfig(cfg).TimeZone(), osCfg.TimeZone())
		require.Equal(t, config.NewApplicationConfig(cfg).Env(), appCfg.Env())
		require.Equal(t, config.NewServerConfig(cfg).Port(), serverCfg.Port())
		require.Equal(t, config.NewMetricsConfig(cfg).Port(), metricsCfg.Port())
		require.Equal(t, config.NewObservabilityConfig(cfg).Enabled(), obsCfg.Enabled())
		require.Equal(t, config.NewDatabaseConfig(cfg).Driver(), dbCfg.Driver())
		require.Equal(t, config.NewDBConnectionConfig(cfg).MaxConns(), dbConnCfg.MaxConns())
		require.Equal(t, config.NewSecurityConfig(cfg).AllowedOrigins(), secCfg.AllowedOrigins())
		require.Equal(t, config.NewIPRateLimitConfig(cfg).Enabled(), ipCfg.Enabled())
	})
}
