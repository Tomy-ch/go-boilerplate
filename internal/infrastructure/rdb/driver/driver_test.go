package driver

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/exaring/otelpgx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracedDB(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("クエリトレーサーを結線したDB接続が生成される", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbCfg.SetDatabaseHost(t, "localhost")
			osCfg := config.NewOperatingSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewTracedDB(dbCfg, osCfg, dbConnCfg, otelpgx.NewTracer())
			require.NoError(t, err)
			require.NotNil(t, db)
			t.Cleanup(func() {
				require.NoError(t, db.Close())
			})

			require.NoError(t, db.Ping(context.Background()))
		})
	})
}

func TestNewDB(t *testing.T) {
	t.Parallel()
	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DB接続が成功する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbCfg.SetDatabaseHost(t, "localhost")
			osCfg := config.NewOperatingSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.NoError(t, err)
			require.NotNil(t, db)

			ctx := context.Background()

			// 疎通確認
			err = db.Ping(ctx)
			require.NoError(t, err)

			ct, err := db.Exec(ctx, "SELECT 1")
			require.NoError(t, err)
			require.NotEmpty(t, ct)

			rows, err := db.Query(ctx, "SELECT 1")
			require.NoError(t, err)
			require.NotNil(t, rows)
			for rows.Next() {
			}
			rows.Close()
			require.NoError(t, rows.Err())

			row := db.QueryRow(ctx, "SELECT 1")
			var n int
			err = row.Scan(&n)
			require.NoError(t, err)

			stat := db.Stats()
			require.NotNil(t, stat)

			require.NoError(t, db.Close())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("DSNが不正な場合、パースに失敗する", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			// 無効なホストを設定して、DSNのパースに失敗させる
			dbCfg.SetDatabaseHost(t, "://invalid-host")
			osCfg := config.NewOperatingSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse DB config")
			require.Nil(t, db)
		})

		t.Run("コネクションプールの作成に失敗する", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			osCfg := config.NewOperatingSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)
			// 無効な値を設定して、コネクションプールの作成に失敗させる
			dbConnCfg.SetMaxConns(t, -1)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			require.Nil(t, db)
			assert.Contains(t, err.Error(), "failed to create DB connection pool")
		})

		t.Run("Pingに失敗する", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbCfg.SetDatabaseName(t, "nonexistentdb")
			osCfg := config.NewOperatingSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			require.Nil(t, db)
			assert.Contains(t, err.Error(), "failed to ping DB")
		})
	})
}
