package migrate

import (
	"errors"
	"math"
	"os"
	"testing"

	mock_migrate "go-boilerplate/internal/cli/migrate/mock"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// errBoom は、テスト用の任意の失敗を表すセンチネルエラーです。
var errBoom = errors.New("boom")

// factoryReturning は、常に与えた migrator を返す migratorFactory を生成します。
func factoryReturning(m migrator) migratorFactory {
	return func(_ string) (migrator, error) { return m, nil }
}

// factoryFailing は、常にエラーを返す migratorFactory を生成します。
func factoryFailing(err error) migratorFactory {
	return func(_ string) (migrator, error) { return nil, err }
}

func TestNewMigrateUpCommand(t *testing.T) {
	t.Parallel()

	cmd := NewMigrateUpCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "migrate-up", cmd.Use)

	steps := cmd.Flags().Lookup("steps")
	require.NotNil(t, steps)
	assert.Equal(t, "0", steps.DefValue)

	database := cmd.Flags().Lookup("database")
	require.NotNil(t, database)
	assert.Empty(t, database.DefValue)

	// 旧 --version フラグは廃止され存在しないこと。
	assert.Nil(t, cmd.Flags().Lookup("version"))
}

func TestNewMigrateDownCommand(t *testing.T) {
	t.Parallel()

	cmd := NewMigrateDownCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "migrate-down", cmd.Use)

	steps := cmd.Flags().Lookup("steps")
	require.NotNil(t, steps)
	assert.Equal(t, "0", steps.DefValue)

	database := cmd.Flags().Lookup("database")
	require.NotNil(t, database)
	assert.Empty(t, database.DefValue)

	assert.Nil(t, cmd.Flags().Lookup("version"))
}

func TestMigrateUpRun(t *testing.T) {
	t.Parallel()

	t.Run("異常系_負のステップ数は検証で弾かれfactoryを呼ばない", func(t *testing.T) {
		t.Parallel()

		called := false
		factory := func(_ string) (migrator, error) {
			called = true
			return nil, nil
		}

		err := migrateUpRun(-1, "", factory)
		require.Error(t, err)
		assert.False(t, called)
	})

	t.Run("異常系_migratorの生成に失敗した場合はそのエラーを返す", func(t *testing.T) {
		t.Parallel()

		err := migrateUpRun(0, "", factoryFailing(errBoom))
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("正常系_ステップ未指定なら全件Upを実行する", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Up().Return(nil)

		err := migrateUpRun(0, "", factoryReturning(m))
		require.NoError(t, err)
	})

	t.Run("正常系_全件Upで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Up().Return(migrate.ErrNoChange)

		err := migrateUpRun(0, "", factoryReturning(m))
		require.NoError(t, err)
	})

	t.Run("異常系_全件Upが失敗した場合はそのエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Up().Return(errBoom)

		err := migrateUpRun(0, "", factoryReturning(m))
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("正常系_正のステップ数なら段数指定でUpを実行する", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Steps(2).Return(nil)

		err := migrateUpRun(2, "", factoryReturning(m))
		require.NoError(t, err)
	})

	t.Run("正常系_段数指定Upで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Steps(1).Return(migrate.ErrNoChange)

		err := migrateUpRun(1, "", factoryReturning(m))
		require.NoError(t, err)
	})

	t.Run("異常系_段数指定Upが失敗した場合はそのエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Steps(3).Return(errBoom)

		err := migrateUpRun(3, "", factoryReturning(m))
		require.ErrorIs(t, err, errBoom)
	})
}

func TestMigrateDownRun(t *testing.T) {
	t.Parallel()

	t.Run("異常系_負のステップ数は検証で弾かれUp方向への巻き戻しを防ぐ", func(t *testing.T) {
		t.Parallel()

		called := false
		factory := func(_ string) (migrator, error) {
			called = true
			return nil, nil
		}

		err := migrateDownRun(-2, "", factory)
		require.Error(t, err)
		assert.False(t, called)
	})

	t.Run("異常系_migratorの生成に失敗した場合はそのエラーを返す", func(t *testing.T) {
		t.Parallel()

		err := migrateDownRun(0, "", factoryFailing(errBoom))
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("正常系_ステップ未指定なら全件Downを実行する", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Version().Return(uint(3), false, nil)
		m.EXPECT().Down().Return(nil)

		err := migrateDownRun(0, "", factoryReturning(m))
		require.NoError(t, err)
	})

	t.Run("異常系_全件Downが失敗した場合はそのエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Version().Return(uint(3), false, nil)
		m.EXPECT().Down().Return(errBoom)

		err := migrateDownRun(0, "", factoryReturning(m))
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("正常系_正のステップ数なら負数へ反転してDownを実行する", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Steps(-2).Return(nil)

		err := migrateDownRun(2, "", factoryReturning(m))
		require.NoError(t, err)
	})

	t.Run("正常系_段数指定Downで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Steps(-1).Return(migrate.ErrNoChange)

		err := migrateDownRun(1, "", factoryReturning(m))
		require.NoError(t, err)
	})

	t.Run("異常系_段数指定Downが失敗した場合はそのエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Steps(-3).Return(errBoom)

		err := migrateDownRun(3, "", factoryReturning(m))
		require.ErrorIs(t, err, errBoom)
	})
}

func TestExecuteMigrateFullDown(t *testing.T) {
	t.Parallel()

	t.Run("正常系_NilVersionは未適用として扱いそのままDownする", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Version().Return(uint(0), false, migrate.ErrNilVersion)
		m.EXPECT().Down().Return(nil)

		require.NoError(t, executeMigrateFullDown(m))
	})

	t.Run("異常系_Version取得がErrNilVersion以外で失敗した場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Version().Return(uint(0), false, errBoom)

		require.ErrorIs(t, executeMigrateFullDown(m), errBoom)
	})

	t.Run("正常系_dirty時はForceで整合を取り直してからDownする", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Version().Return(uint(5), true, nil)
		m.EXPECT().Force(5).Return(nil)
		m.EXPECT().Down().Return(nil)

		require.NoError(t, executeMigrateFullDown(m))
	})

	t.Run("異常系_dirty時のForceが失敗した場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Version().Return(uint(5), true, nil)
		m.EXPECT().Force(5).Return(errBoom)

		require.ErrorIs(t, executeMigrateFullDown(m), errBoom)
	})

	t.Run("異常系_dirty時にバージョンがintへ変換できない場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		// MaxInt を超える uint は safecast でオーバーフロー検出され、Force/Down へ進まない。
		m.EXPECT().Version().Return(uint(math.MaxUint64), true, nil)

		require.Error(t, executeMigrateFullDown(m))
	})

	t.Run("正常系_Downが無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := mock_migrate.NewMockmigrator(ctrl)
		m.EXPECT().Version().Return(uint(2), false, nil)
		m.EXPECT().Down().Return(migrate.ErrNoChange)

		require.NoError(t, executeMigrateFullDown(m))
	})
}

func TestOverrideEnv(t *testing.T) {
	// 環境変数を触るため、この関数は t.Parallel を使わない。

	t.Run("正常系_既存値は復元され未設定値はUnsetされる", func(t *testing.T) {
		// 環境変数を触るため、このサブテストは並行化しない。
		const existingKey = "TEST_OVERRIDE_EXISTING"
		const absentKey = "TEST_OVERRIDE_ABSENT"

		t.Setenv(existingKey, "original")

		restoreExisting := overrideEnv(existingKey, "changed")
		v, ok := os.LookupEnv(existingKey)
		require.True(t, ok)
		assert.Equal(t, "changed", v)
		restoreExisting()
		v, ok = os.LookupEnv(existingKey)
		require.True(t, ok)
		assert.Equal(t, "original", v)

		restoreAbsent := overrideEnv(absentKey, "temp")
		v, ok = os.LookupEnv(absentKey)
		require.True(t, ok)
		assert.Equal(t, "temp", v)
		restoreAbsent()
		_, ok = os.LookupEnv(absentKey)
		assert.False(t, ok)
	})
}
