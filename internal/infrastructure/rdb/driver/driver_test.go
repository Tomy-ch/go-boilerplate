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

			ctx := context.Background()

			// 疎通確認
			err = db.Ping(ctx)
			require.NoError(t, err)

			ct, err := db.Exec(ctx, "SELECT 1")
			require.NotEmpty(t, ct)
			require.NoError(t, err)

			rows, err := db.Query(ctx, "SELECT 1")
			require.NoError(t, err)
			require.NotNil(t, rows)
			for rows.Next() {
			}
			rows.Close()

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
			// パスワードに特殊文字を含めて、接続文字列の生成に失敗させる
			dbCfg.SetDatabasePassword(t, "p%ZZword")
			osCfg := config.NewOperationSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			require.Nil(t, db)
			require.Contains(t, err.Error(), "failed to parse DB config")
		})

		t.Run("コネクションプールの作成に失敗する", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			osCfg := config.NewOperationSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)
			// 無効な値を設定して、コネクションプールの作成に失敗させる
			dbConnCfg.SetMaxOpenConns(t, -1)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			require.Nil(t, db)
			require.Contains(t, err.Error(), "failed to create DB connection pool")
		})

		t.Run("Pingに失敗する", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			dbCfg := config.NewDatabaseConfig(cfg)
			dbCfg.SetDatabaseName(t, "nonexistentdb")
			osCfg := config.NewOperationSystemConfig(cfg)
			dbConnCfg := config.NewDBConnectionConfig(cfg)

			db, err := NewDB(dbCfg, osCfg, dbConnCfg)
			require.Error(t, err)
			require.Nil(t, db)
			require.Contains(t, err.Error(), "failed to ping DB")
		})
	})
}
