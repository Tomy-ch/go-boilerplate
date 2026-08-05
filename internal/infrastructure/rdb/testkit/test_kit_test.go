package testkit

import (
	"context"
	"testing"

	"go-boilerplate/internal/config"
	"go-boilerplate/internal/logging"

	"go-boilerplate/internal/infrastructure/rdb/driver"
	"go-boilerplate/internal/infrastructure/system"
	mock_tx "go-boilerplate/internal/usecase/boundary/tx/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func TestNewTestDB(t *testing.T) {
	t.Parallel()
	db := NewTestDB(t)
	// 返る DB が生きている（接続可能）ことを検証する。
	require.NoError(t, db.Ping(context.Background()))
}

func TestNewTestTransactionRunner(t *testing.T) {
	t.Parallel()
	runner := NewTestTransactionRunner(t)
	// 公開 API 経由で WithinTx がコールバックを実行する（実トランザクションを開始しロールバックする）ことを検証する。
	ran := false
	runner.WithinTx(func(context.Context) { ran = true })
	assert.True(t, ran)
}

func Test_testTxRunner_WithinTx(t *testing.T) {
	t.Parallel()
	cfg := config.MockConfigForTest(t)
	dbCfg := config.NewDatabaseConfig(cfg)
	osCfg := config.NewOperatingSystemConfig(cfg)
	dbConnCfg := config.NewDBConnectionConfig(cfg)

	testLogger := logging.NewTestLogger(t)

	db, err := driver.NewDB(dbCfg, osCfg, dbConnCfg)
	require.NoError(t, err)
	innerTxm := driver.NewTransactionManager(db, dbCfg, testLogger, system.NewSleeper())

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("実行時にエラーが発生しない場合、正常に終了すること", func(t *testing.T) {
			t.Parallel()
			txm := &testTxRunner{
				inner: innerTxm,
				db:    db,
				t:     t,
			}
			txm.WithinTx(func(_ context.Context) {})
		})

		t.Run("Doがロールバックsentinel以外のnilを返す場合、NoError検証まで到達すること", func(t *testing.T) {
			t.Parallel()
			// inner.Do が nil を返すと、rollback sentinel 判定を外れて require.NoError の検証経路に到達する。
			ctrl := gomock.NewController(t)
			manager := mock_tx.NewMockManager(ctrl)
			manager.EXPECT().Do(gomock.Any(), gomock.Any()).Return(nil)

			txm := &testTxRunner{
				inner: manager,
				t:     t,
			}
			txm.WithinTx(func(_ context.Context) {})
		})
	})
}

func TestHoldSuiteSerialization(t *testing.T) {
	t.Parallel()

	db := NewTestDB(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("占有中は他セッションが同じキーを取得できない", func(t *testing.T) {
			t.Parallel()
			HoldSuiteSerialization(t, db)

			// 直列化の実体は advisory lock の排他性なので、別セッション（接続プール直）から
			// 同じキーを取れないことで検証する。取れてしまうなら並行実行を止められていない。
			ctx := context.Background()
			var acquired bool
			row := driver.New(ctx, db).QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", txAdvisoryLockKey)
			require.NoError(t, row.Scan(&acquired))
			assert.False(t, acquired)
		})
	})
}

func Test_lockSuiteSerialization(t *testing.T) {
	t.Parallel()

	db := NewTestDB(t)
	runner := NewTestTransactionRunner(t)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("同一トランザクションからの再取得は待たずに成立する", func(t *testing.T) {
			t.Parallel()
			// WithinTx が既に同じキーを保持している。advisory lock は同一トランザクション内で再入可能。
			runner.WithinTx(func(ctx context.Context) {
				require.NoError(t, lockSuiteSerialization(ctx, driver.New(ctx, db)))
			})
		})
	})
}

func Test_getTestDB(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続可能なドライバを返す", func(t *testing.T) {
			t.Parallel()

			db := getTestDB(t)

			require.NotNil(t, db)
			require.NoError(t, db.Ping(context.Background()))
		})

		t.Run("複数回呼び出しても同一のドライバを共有する", func(t *testing.T) {
			t.Parallel()

			first := getTestDB(t)
			second := getTestDB(t)

			assert.Same(t, first, second)
		})
	})
}
