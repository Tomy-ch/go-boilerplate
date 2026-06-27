package driver

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_applyDBTimeouts(t *testing.T) {
	t.Parallel()

	newPoolCfg := func(t *testing.T) *pgxpool.Config {
		t.Helper()
		cfg, err := pgxpool.ParseConfig("postgres://u:p@localhost:5432/db")
		require.NoError(t, err)
		return cfg
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正の値はミリ秒のRuntimeParamsとして設定される", func(t *testing.T) {
			t.Parallel()

			cfg := newPoolCfg(t)
			applyDBTimeouts(cfg, 30*time.Second, 10*time.Second)

			assert.Equal(t, "30000", cfg.ConnConfig.RuntimeParams["statement_timeout"])
			assert.Equal(t, "10000", cfg.ConnConfig.RuntimeParams["lock_timeout"])
		})

		t.Run("0以下は設定されない_Postgres既定の無制限のまま", func(t *testing.T) {
			t.Parallel()

			cfg := newPoolCfg(t)
			applyDBTimeouts(cfg, 0, -1)

			assert.NotContains(t, cfg.ConnConfig.RuntimeParams, "statement_timeout")
			assert.NotContains(t, cfg.ConnConfig.RuntimeParams, "lock_timeout")
		})
	})
}
