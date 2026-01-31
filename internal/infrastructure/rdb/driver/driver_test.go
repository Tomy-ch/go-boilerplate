package driver

import (
	"context"
	"testing"
	"time"

	"boilerplate-go/internal/config"
	"boilerplate-go/internal/ctxhelper"

	"github.com/stretchr/testify/require"
)

func TestNewDB(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DB接続が成功する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbCfg.SetDatabaseHost(t, "localhost")
			osCfg := config.NewOperationSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.NoError(t, err)
			require.NotNil(t, db)

			// 疎通確認
			err = db.PingContext(context.Background())
			require.NoError(t, err)

			err = db.Close()
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DSNが無効", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbCfg.SetDatabaseDriver(t, "invalid_driver")
			osCfg := config.NewOperationSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			require.Nil(t, db)
		})

		t.Run("Pingに失敗", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbCfg.SetDatabaseName(t, "nonexistentdb")
			osCfg := config.NewOperationSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			require.Nil(t, db)
		})
	})
}

func Test_dbDriver_ResolveQueryTimeout(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	expectedDelta := float64(1 * time.Millisecond)

	t.Run("ctxにDBタイムアウトのオーバーライドが設定されている場合、その値でWithTimeoutされる", func(t *testing.T) {
		defaultTimeout := 3 * time.Second
		dbCfg := config.NewDatabaseConfig(cfg)
		dbCfg.SetDefaultTimeout(t, defaultTimeout)
		override := 1 * time.Second

		mockDB := NewMockInstance(t)
		mockDBDriver := mockDB.(*dbDriver)
		mockDBDriver.dbCfg = dbCfg

		baseCtx := context.Background()
		ctx := ctxhelper.WithDbTimeout(baseCtx, override)

		gotCtx, cancel := mockDBDriver.ResolveQueryTimeout(ctx)
		defer cancel()

		deadline, ok := gotCtx.Deadline()
		require.True(t, ok, "Deadline should be set when override > 0")

		now := time.Now()
		diff := deadline.Sub(now)

		require.InDelta(t, float64(override), float64(diff), expectedDelta)
	})

	t.Run("ctxに0以下のDBタイムアウトが設定されている場合、タイムアウトなしとしてそのctxをそのまま返す", func(t *testing.T) {
		t.Parallel()

		defaultTimeout := 3 * time.Second
		dbCfg := config.NewDatabaseConfig(cfg)
		dbCfg.SetDefaultTimeout(t, defaultTimeout)
		override := time.Duration(0)

		mockDB := NewMockInstance(t)
		mockDBDriver := mockDB.(*dbDriver)
		mockDBDriver.dbCfg = config.NewDatabaseConfig(config.MockConfigForTest(t))

		baseCtx := context.Background()
		ctx := ctxhelper.WithDbTimeout(baseCtx, override)

		gotCtx, cancel := mockDBDriver.ResolveQueryTimeout(ctx)
		defer cancel()

		require.Same(t, ctx, gotCtx)

		_, ok := gotCtx.Deadline()
		require.False(t, ok)
	})

	t.Run("ctxにDBタイムアウトのオーバーライドが無い場合、デフォルトタイムアウトでWithTimeoutされる", func(t *testing.T) {
		t.Parallel()

		defaultTimeout := 3 * time.Second
		dbCfg := config.NewDatabaseConfig(cfg)
		dbCfg.SetDefaultTimeout(t, defaultTimeout)

		mockDB := NewMockInstance(t)
		mockDBDriver := mockDB.(*dbDriver)
		mockDBDriver.dbCfg = dbCfg

		baseCtx := context.Background()

		gotCtx, cancel := mockDBDriver.ResolveQueryTimeout(baseCtx)
		defer cancel()

		deadline, ok := gotCtx.Deadline()
		require.True(t, ok)

		now := time.Now()
		diff := deadline.Sub(now)

		require.InDelta(t, float64(defaultTimeout), float64(diff), expectedDelta)
	})
}
