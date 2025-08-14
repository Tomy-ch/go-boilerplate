package rdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

func TestNewTransactionManager(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	db, err := sql.Open("pgx", cfg.DatabaseDSN())
	require.NoError(t, err)
	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	manager := NewTransactionManager(&cfg, db)
	require.NotNil(t, manager)
}

func TestTxManager_Do(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	cfg.SetDatabaseHost(t, "localhost")
	db, err := sql.Open("pgx", cfg.DatabaseDSN())
	require.NoError(t, err)
	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	manager := NewTransactionManager(&cfg, db)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションがコミットされる", func(t *testing.T) {
			ctx := context.Background()
			err := manager.Do(ctx, func(_ context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("トランザクションがロールバックされる", func(t *testing.T) {
			ctx := context.Background()
			err := manager.Do(ctx, func(_ context.Context) error {
				return errors.New("rollback")
			})
			require.Error(t, err)
			require.EqualError(t, err, "rollback")
		})

		t.Run("パニックが発生した場合にロールバックされる", func(t *testing.T) {
			ctx := context.Background()
			defer func() {
				r := recover()
				require.NotNil(t, r)
			}()

			_ = manager.Do(ctx, func(_ context.Context) error {
				panic("panic occurred")
			})
		})

		t.Run("すでにtxがある場合", func(t *testing.T) {
			ctx := context.Background()
			tx, err := db.BeginTx(ctx, nil)
			require.NoError(t, err)
			t.Cleanup(func() {
				err = tx.Rollback()
				require.NoError(t, err)
			})

			ctx = withTx(ctx, tx)
			err = manager.Do(ctx, func(_ context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})
	})
}

func TestResolveConn(t *testing.T) {
	cfg := config.MockConfigForTest(t)
	cfg.SetDatabaseHost(t, "localhost")
	db, err := sql.Open("pgx", cfg.DatabaseDSN())
	require.NoError(t, err)
	t.Cleanup(func() {
		err := db.Close()
		require.NoError(t, err)
	})

	t.Run("トランザクションが存在する場合", func(t *testing.T) {
		tx, err := db.BeginTx(context.Background(), nil)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := tx.Rollback()
			require.NoError(t, err)
		})

		ctx := withTx(context.Background(), tx)
		conn := ResolveConn(ctx, db)
		require.Equal(t, tx, conn)
	})

	t.Run("トランザクションが存在しない場合", func(t *testing.T) {
		ctx := context.Background()
		conn := ResolveConn(ctx, db)
		require.Equal(t, db, conn)
	})
}
