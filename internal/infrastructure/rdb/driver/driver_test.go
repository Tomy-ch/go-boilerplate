package driver

import (
	"context"
	"testing"

	"boilerplate-go/internal/config"

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
