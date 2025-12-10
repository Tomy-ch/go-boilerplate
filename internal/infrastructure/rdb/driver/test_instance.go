package driver

import (
	"database/sql"
	"testing"

	"boilerplate-go/internal/config"
	"boilerplate-go/pkg/ptr"

	"github.com/stretchr/testify/require"
)

// NewMockInstance は、テスト用のDatabaseDriverインスタンスを生成します。
func NewMockInstance(t *testing.T) DatabaseDriver {
	t.Helper()
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	return &dbDriver{
		db:    ptr.To(sql.DB{}),
		dbCfg: dbCfg,
	}
}

func NewTestInstance(t *testing.T, dbCfg *config.DatabaseConfig) DatabaseDriver {
	t.Helper()
	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOSConfig(cfg)

	db, err := sql.Open("pgx", dbCfg.DSN(osCfg))
	require.NoError(t, err, "failed to open database")

	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err, "failed to close database")
	})

	return &dbDriver{
		db:    db,
		dbCfg: dbCfg,
	}
}
