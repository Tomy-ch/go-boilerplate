package migrate

import (
	"math"
	"testing"

	mock_migrate "go-boilerplate/internal/cli/migrate/mock"
	"go-boilerplate/internal/logging"
	"go-boilerplate/pkg/xerrors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// errBoom は、テスト用の任意の失敗を表すセンチネルエラーです。
var errBoom = xerrors.New("boom")

// factoryReturning は、常に与えた Migrator を返す MigratorFactory を生成します。
func factoryReturning(m Migrator) MigratorFactory {
	return func(_ string) (Migrator, error) { return m, nil }
}

// factoryFailing は、常にエラーを返す MigratorFactory を生成します。
func factoryFailing(err error) MigratorFactory {
	return func(_ string) (Migrator, error) { return nil, err }
}

func TestMigrateUpRun(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステップ未指定なら全件Upを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Up().Return(nil)

			err := MigrateUpRun(0, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})

		t.Run("全件Upで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Up().Return(migrate.ErrNoChange)

			err := MigrateUpRun(0, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})

		t.Run("正のステップ数なら段数指定でUpを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(2).Return(nil)

			err := MigrateUpRun(2, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})

		t.Run("段数指定Upで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(1).Return(migrate.ErrNoChange)

			err := MigrateUpRun(1, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("負のステップ数は検証で弾かれfactoryを呼ばない", func(t *testing.T) {
			t.Parallel()

			called := false
			factory := func(_ string) (Migrator, error) {
				called = true
				//nolint:nilnil // テスト用スタブのため意図的にnil,nilを返す
				return nil, nil
			}

			err := MigrateUpRun(-1, "", logging.NewTestLogger(t), factory)
			require.ErrorIs(t, err, errNegativeSteps)
			assert.False(t, called)
		})

		t.Run("migratorの生成に失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			err := MigrateUpRun(0, "", logging.NewTestLogger(t), factoryFailing(errBoom))
			require.ErrorIs(t, err, errBoom)
		})

		t.Run("全件Upが失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Up().Return(errBoom)

			err := MigrateUpRun(0, "", logging.NewTestLogger(t), factoryReturning(m))
			require.ErrorIs(t, err, errBoom)
		})

		t.Run("段数指定Upが失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(3).Return(errBoom)

			err := MigrateUpRun(3, "", logging.NewTestLogger(t), factoryReturning(m))
			require.ErrorIs(t, err, errBoom)
		})
	})
}

func TestMigrateDownRun(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステップ未指定なら全件Downを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(3), false, nil)
			m.EXPECT().Down().Return(nil)

			err := MigrateDownRun(0, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})

		t.Run("ステップ未指定でdirty状態ならForceで整合を取ってから全件Downする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(4), true, nil)
			m.EXPECT().Force(4).Return(nil)
			m.EXPECT().Down().Return(nil)

			err := MigrateDownRun(0, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})

		t.Run("正のステップ数なら負数へ反転してDownを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(-2).Return(nil)

			err := MigrateDownRun(2, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})

		t.Run("段数指定Downで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(-1).Return(migrate.ErrNoChange)

			err := MigrateDownRun(1, "", logging.NewTestLogger(t), factoryReturning(m))
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("負のステップ数は検証で弾かれUp方向への巻き戻しを防ぐ", func(t *testing.T) {
			t.Parallel()

			called := false
			factory := func(_ string) (Migrator, error) {
				called = true
				//nolint:nilnil // テスト用スタブのため意図的にnil,nilを返す
				return nil, nil
			}

			err := MigrateDownRun(-2, "", logging.NewTestLogger(t), factory)
			require.ErrorIs(t, err, errNegativeSteps)
			assert.False(t, called)
		})

		t.Run("migratorの生成に失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			err := MigrateDownRun(0, "", logging.NewTestLogger(t), factoryFailing(errBoom))
			require.ErrorIs(t, err, errBoom)
		})

		t.Run("全件Downが失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(3), false, nil)
			m.EXPECT().Down().Return(errBoom)

			err := MigrateDownRun(0, "", logging.NewTestLogger(t), factoryReturning(m))
			require.ErrorIs(t, err, errBoom)
		})

		t.Run("段数指定Downが失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(-3).Return(errBoom)

			err := MigrateDownRun(3, "", logging.NewTestLogger(t), factoryReturning(m))
			require.ErrorIs(t, err, errBoom)
		})
	})
}

func Test_executeMigrateUp(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ステップ未指定なら全件Upを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Up().Return(nil)

			require.NoError(t, executeMigrateUp(m, 0))
		})

		t.Run("全件Upで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Up().Return(migrate.ErrNoChange)

			require.NoError(t, executeMigrateUp(m, 0))
		})

		t.Run("正のステップ数なら段数指定でUpを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(2).Return(nil)

			require.NoError(t, executeMigrateUp(m, 2))
		})

		t.Run("段数指定Upで無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(1).Return(migrate.ErrNoChange)

			require.NoError(t, executeMigrateUp(m, 1))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("全件Upが失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Up().Return(errBoom)

			require.ErrorIs(t, executeMigrateUp(m, 0), errBoom)
		})

		t.Run("段数指定Upが失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(3).Return(errBoom)

			require.ErrorIs(t, executeMigrateUp(m, 3), errBoom)
		})
	})
}

func Test_executeMigrateDownSteps(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("正のステップ数なら負数へ反転してDownを実行する", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(-2).Return(nil)

			require.NoError(t, executeMigrateDownSteps(m, 2))
		})

		t.Run("無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(-1).Return(migrate.ErrNoChange)

			require.NoError(t, executeMigrateDownSteps(m, 1))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Downが失敗した場合はそのエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Steps(-3).Return(errBoom)

			require.ErrorIs(t, executeMigrateDownSteps(m, 3), errBoom)
		})
	})
}

func Test_executeMigrateFullDown(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("NilVersionは未適用として扱いそのままDownする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(0), false, migrate.ErrNilVersion)
			m.EXPECT().Down().Return(nil)

			require.NoError(t, executeMigrateFullDown(m))
		})

		t.Run("dirty時はForceで整合を取り直してからDownする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(5), true, nil)
			m.EXPECT().Force(5).Return(nil)
			m.EXPECT().Down().Return(nil)

			require.NoError(t, executeMigrateFullDown(m))
		})

		t.Run("Downが無変更ならErrNoChangeを握りつぶし成功扱いとする", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(2), false, nil)
			m.EXPECT().Down().Return(migrate.ErrNoChange)

			require.NoError(t, executeMigrateFullDown(m))
		})

		t.Run("dirty時にForce後のDownがErrNoChangeでも成功扱いとする", func(t *testing.T) {
			t.Parallel()

			// dirty=true → Force 成功 → Down(ErrNoChange) の経路で
			// ErrNoChange 握りつぶしが Force 後にも効くことを検証する。
			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(3), true, nil)
			m.EXPECT().Force(3).Return(nil)
			m.EXPECT().Down().Return(migrate.ErrNoChange)

			require.NoError(t, executeMigrateFullDown(m))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Version取得がErrNilVersion以外で失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(0), false, errBoom)

			require.ErrorIs(t, executeMigrateFullDown(m), errBoom)
		})

		t.Run("dirty時のForceが失敗した場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			m.EXPECT().Version().Return(uint(5), true, nil)
			m.EXPECT().Force(5).Return(errBoom)

			require.ErrorIs(t, executeMigrateFullDown(m), errBoom)
		})

		t.Run("dirty時にバージョンがintへ変換できない場合はエラーを返す", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := mock_migrate.NewMockMigrator(ctrl)
			// MaxInt を超える uint は safecast でオーバーフロー検出され、Force/Down へ進まない。
			m.EXPECT().Version().Return(uint(math.MaxUint64), true, nil)

			require.Error(t, executeMigrateFullDown(m))
		})
	})
}
