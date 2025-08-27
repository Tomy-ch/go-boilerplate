package rdbdriver

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

		t.Run("正常系: DB接続が成功する", func(t *testing.T) {
			t.Parallel()

			cfg := config.MockConfigForTest(t)
			cfg.SetDatabaseHost(t, "localhost")

			db, err := NewDB(cfg)
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
			cfg.SetDatabaseDriver(t, "invalid_driver")

			db, err := NewDB(cfg)
			require.Error(t, err)
			require.Nil(t, db)
		})

		t.Run("Pingに失敗", func(t *testing.T) {
			t.Parallel()
			cfg := config.MockConfigForTest(t)
			cfg.SetDatabaseName(t, "nonexistentdb")

			db, err := NewDB(cfg)
			require.Error(t, err)
			require.Nil(t, db)
		})
	})
}
