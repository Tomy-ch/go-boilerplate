package driver

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"go-boilerplate/internal/config"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 接続数の消費を抑えるため、dbDriver のメソッド検証はプールを 1 つだけ共有します
// （閉じる検証だけは newTestDriver で専用プールを用意します）。
var (
	sharedDriverOnce    sync.Once
	sharedDriver        DatabaseDriver
	errSharedDriverInit error
)

// recordingQueryTracer は、クエリ計装が結線されたかを呼び出し回数で観測する pgx.QueryTracer です。
type recordingQueryTracer struct{ started atomic.Int32 }

func (r *recordingQueryTracer) TraceQueryStart(
	ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData,
) context.Context {
	r.started.Add(1)
	return ctx
}

func (r *recordingQueryTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// sharedTestDriver は、パッケージ内で共有するローカル DB 接続を返します。
func sharedTestDriver(t *testing.T) DatabaseDriver {
	t.Helper()

	sharedDriverOnce.Do(func() {
		cfg := config.MockConfigForTest(t)
		dbCfg := config.NewDatabaseConfig(cfg)
		dbCfg.SetDatabaseHost(t, "localhost")
		sharedDriver, errSharedDriverInit = NewDB(
			dbCfg, config.NewOperatingSystemConfig(cfg), config.NewDBConnectionConfig(cfg))
	})

	require.NoError(t, errSharedDriverInit)
	require.NotNil(t, sharedDriver)
	return sharedDriver
}

// newTestDriver は、呼び出し側が閉じる前提の専用プールを持つローカル DB 接続を返します。
func newTestDriver(t *testing.T) DatabaseDriver {
	t.Helper()

	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	dbCfg.SetDatabaseHost(t, "localhost")
	db, err := NewDB(dbCfg, config.NewOperatingSystemConfig(cfg), config.NewDBConnectionConfig(cfg))
	require.NoError(t, err)
	return db
}

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

func Test_newDB(t *testing.T) {
	t.Parallel()

	newConfigs := func(t *testing.T) (
		*config.DatabaseConfig, *config.OperatingSystemConfig, *config.DBConnectionConfig,
	) {
		t.Helper()
		cfg := config.MockConfigForTest(t)
		dbCfg := config.NewDatabaseConfig(cfg)
		dbCfg.SetDatabaseHost(t, "localhost")
		return dbCfg, config.NewOperatingSystemConfig(cfg), config.NewDBConnectionConfig(cfg)
	}

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("tracer が非nilの場合はクエリ計装を接続へ結線する", func(t *testing.T) {
			t.Parallel()

			dbCfg, osCfg, dbConnCfg := newConfigs(t)
			tracer := &recordingQueryTracer{}

			db, err := newDB(dbCfg, osCfg, dbConnCfg, tracer)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })

			_, err = db.Exec(context.Background(), "SELECT 1")
			require.NoError(t, err)
			assert.Positive(t, tracer.started.Load())
		})

		t.Run("tracer が nil の場合は計装を結線しない", func(t *testing.T) {
			t.Parallel()

			dbCfg, osCfg, dbConnCfg := newConfigs(t)

			db, err := newDB(dbCfg, osCfg, dbConnCfg, nil)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })

			drv, ok := db.(*dbDriver)
			require.True(t, ok)
			assert.Nil(t, drv.pool.Config().ConnConfig.Tracer)
		})
	})
}

func Test_dbDriver_Exec(t *testing.T) {
	t.Parallel()

	db := sharedTestDriver(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SQLを実行し実行結果のコマンドタグを返す", func(t *testing.T) {
			t.Parallel()

			ct, err := db.Exec(context.Background(), "SELECT 1")

			require.NoError(t, err)
			assert.Equal(t, "SELECT 1", ct.String())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実行に失敗した場合はPostgreSQLのエラーを正規化せずそのまま返す", func(t *testing.T) {
			t.Parallel()

			_, err := db.Exec(context.Background(), "SELECT 1 FROM table_that_does_not_exist")

			require.Error(t, err)
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			assert.Equal(t, "42P01", pgErr.Code) // undefined_table
		})
	})
}

func Test_dbDriver_Query(t *testing.T) {
	t.Parallel()

	db := sharedTestDriver(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数行の結果を順に読み出せる", func(t *testing.T) {
			t.Parallel()

			rows, err := db.Query(context.Background(), "SELECT n FROM generate_series(1, 3) AS n")
			require.NoError(t, err)
			t.Cleanup(rows.Close)

			var got []int
			for rows.Next() {
				var n int
				require.NoError(t, rows.Scan(&n))
				got = append(got, n)
			}
			require.NoError(t, rows.Err())
			assert.Equal(t, []int{1, 2, 3}, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実行に失敗した場合はPostgreSQLのエラーを正規化せずそのまま返す", func(t *testing.T) {
			t.Parallel()

			//nolint:sqlclosecheck // 失敗時は rows が nil で返るため close 対象が存在しない
			_, err := db.Query(context.Background(), "SELECT 1 FROM table_that_does_not_exist")

			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			assert.Equal(t, "42P01", pgErr.Code) // undefined_table
		})
	})
}

func Test_dbDriver_QueryRow(t *testing.T) {
	t.Parallel()

	db := sharedTestDriver(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("単一行の値をスキャンできる", func(t *testing.T) {
			t.Parallel()

			var got int
			require.NoError(t, db.QueryRow(context.Background(), "SELECT 42").Scan(&got))

			assert.Equal(t, 42, got)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("該当行が無い場合はスキャン時にpgx.ErrNoRowsを返す", func(t *testing.T) {
			t.Parallel()

			var got int
			err := db.QueryRow(context.Background(), "SELECT 1 WHERE false").Scan(&got)

			require.ErrorIs(t, err, pgx.ErrNoRows)
		})
	})
}

func Test_dbDriver_Begin(t *testing.T) {
	t.Parallel()

	db := sharedTestDriver(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションを開始しその中でクエリを実行できる", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			tx, err := db.Begin(ctx)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tx.Rollback(ctx)) })

			var got int
			require.NoError(t, tx.QueryRow(ctx, "SELECT 7").Scan(&got))
			assert.Equal(t, 7, got)
		})

		t.Run("開始したトランザクションは互いに独立している", func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			tx1, err := db.Begin(ctx)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tx1.Rollback(ctx)) })

			tx2, err := db.Begin(ctx)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tx2.Rollback(ctx)) })

			assert.NotSame(t, tx1, tx2)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みcontextではトランザクションを開始せずエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			tx, err := db.Begin(ctx)

			require.ErrorIs(t, err, context.Canceled)
			assert.Nil(t, tx)
		})
	})
}

func Test_dbDriver_Ping(t *testing.T) {
	t.Parallel()

	db := sharedTestDriver(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続可能なプールではnilを返す", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, db.Ping(context.Background()))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("キャンセル済みcontextではエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			require.ErrorIs(t, db.Ping(ctx), context.Canceled)
		})
	})
}

func Test_dbDriver_Stats(t *testing.T) {
	t.Parallel()

	db := sharedTestDriver(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("設定した最大接続数を反映した統計を返す", func(t *testing.T) {
			t.Parallel()

			want := config.NewDBConnectionConfig(config.MockConfigForTest(t)).MaxConns()

			stat := db.Stats()

			require.NotNil(t, stat)
			assert.Equal(t, want, stat.MaxConns())
		})

		t.Run("接続を取得するたびに更新される生きた統計を返す", func(t *testing.T) {
			t.Parallel()

			before := db.Stats().AcquireCount()
			require.NoError(t, db.Ping(context.Background()))

			assert.Greater(t, db.Stats().AcquireCount(), before)
		})
	})
}

func Test_dbDriver_Close(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("プールを閉じてnilを返し以降は接続できなくなる", func(t *testing.T) {
			t.Parallel()

			db := newTestDriver(t)

			require.NoError(t, db.Close())

			require.Error(t, db.Ping(context.Background()))
		})
	})
}
