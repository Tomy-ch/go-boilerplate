package fixcollation

import (
	"context"
	"testing"

	"go-boilerplate/internal/logging"
	mock_exec "go-boilerplate/pkg/exec/mock"
	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_validateDatabaseName(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("localは許可", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateDatabaseName("local"))
		})
		t.Run("testは許可", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateDatabaseName("test"))
		})
		t.Run("template1は許可", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateDatabaseName("template1"))
		})
		t.Run("スロットのlocalは許可", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateDatabaseName("wt1_local"))
		})
		t.Run("スロットのtestは許可", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, validateDatabaseName("wt12_test"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("空文字は不許可", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateDatabaseName(""), errInvalidDatabaseName)
		})
		t.Run("想定外のDB名は不許可", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateDatabaseName("production"), errInvalidDatabaseName)
		})
		t.Run("template0は接続できないため不許可", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateDatabaseName("template0"), errInvalidDatabaseName)
		})
		t.Run("スロット番号が0のDB名は不許可", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateDatabaseName("wt0_local"), errInvalidDatabaseName)
		})
		t.Run("スロット名の接尾辞が想定外なら不許可", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateDatabaseName("wt1_prod"), errInvalidDatabaseName)
		})
		t.Run("SQLを差し込む形のDB名は不許可", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateDatabaseName("local; DROP DATABASE test"), errInvalidDatabaseName)
		})
	})
}

func Test_withDatabase(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("接続先データベースを指定した名前へ差し替える", func(t *testing.T) {
			t.Parallel()

			actual, err := withDatabase("postgres://user@localhost:5432/local?sslmode=disable", "template1")

			require.NoError(t, err)
			assert.Equal(t, "postgres://user@localhost:5432/template1?sslmode=disable", actual)
		})

		t.Run("ホストとクエリパラメータは保持する", func(t *testing.T) {
			t.Parallel()

			actual, err := withDatabase("postgres://user@db.example.com:6543/wt1_local?sslmode=require", "wt3_test")

			require.NoError(t, err)
			assert.Equal(t, "postgres://user@db.example.com:6543/wt3_test?sslmode=require", actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("解釈できない接続URLはErrInvalidDSNを返す", func(t *testing.T) {
			t.Parallel()

			actual, err := withDatabase("postgres://user@localhost:5432/local\x7f", "local")

			assert.Empty(t, actual)
			require.ErrorIs(t, err, errInvalidDSN)
		})
	})
}

func Test_fixCollation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("REINDEXとALTERをこの順序とON_ERROR_STOP指定で実行する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runner := mock_exec.NewMockRunner(ctrl)

			// 2 文 (REINDEX → ALTER) が psql 経由で順序通り実行され、 ON_ERROR_STOP=1 が常に
			// 付与され、 SQL 内容自体がデータベース名込みで一致することを検証する。
			gomock.InOrder(
				runner.EXPECT().Output(gomock.Any(), workDir, gomock.Any(), psqlCommand,
					[]string{"postgres://dsn", "-v", "ON_ERROR_STOP=1", "-c", "REINDEX DATABASE local;"}).
					Return(nil, nil),
				runner.EXPECT().Output(gomock.Any(), workDir, gomock.Any(), psqlCommand,
					[]string{"postgres://dsn", "-v", "ON_ERROR_STOP=1", "-c", "ALTER DATABASE local REFRESH COLLATION VERSION;"}).
					Return(nil, nil),
			)

			err := fixCollation(context.Background(), runner, logging.NewTestLogger(t), "postgres://dsn", "pw", "local")
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("1文目のREINDEXが失敗するとエラーを返し2文目は実行しない", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runner := mock_exec.NewMockRunner(ctrl)

			// 1 文目で失敗 → 2 文目は実行されない（Times(1)）。
			runner.EXPECT().
				Output(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, xerrors.New("psql failed")).
				Times(1)

			err := fixCollation(context.Background(), runner, logging.NewTestLogger(t), "postgres://dsn", "pw", "local")
			require.Error(t, err)
		})

		t.Run("REINDEX成功後の2文目ALTERが失敗するとエラーを返す", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runner := mock_exec.NewMockRunner(ctrl)

			// 1 文目成功 → 2 文目失敗。 ループの 2 周目エラー return path を検証。
			gomock.InOrder(
				runner.EXPECT().Output(gomock.Any(), workDir, gomock.Any(), psqlCommand, gomock.Any()).
					Return(nil, nil),
				runner.EXPECT().Output(gomock.Any(), workDir, gomock.Any(), psqlCommand, gomock.Any()).
					Return(nil, xerrors.New("alter failed")),
			)

			err := fixCollation(context.Background(), runner, logging.NewTestLogger(t), "postgres://dsn", "pw", "local")
			require.Error(t, err)
		})
	})
}

func TestRunFix(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("検証通過後にDSNを解決しcollation修正を実行する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runner := mock_exec.NewMockRunner(ctrl)
			runner.EXPECT().Output(gomock.Any(), workDir, gomock.Any(), psqlCommand, gomock.Any()).Return(nil, nil).Times(2)

			loadDSN := func() (string, string, error) { return "postgres://dsn", "pw", nil }
			err := RunFix(context.Background(), runner, logging.NewTestLogger(t), "local", loadDSN)
			require.NoError(t, err)
		})

		t.Run("設定の接続先ではなく指定したデータベースへ接続する", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runner := mock_exec.NewMockRunner(ctrl)

			// REINDEX DATABASE は接続中のデータベースしか対象にできないため、設定が local を指していても
			// 指定した template1 へ接続し直すことを、psql へ渡す接続 URL で固定する。
			const targetURL = "postgres://user@localhost:5432/template1?sslmode=disable"
			gomock.InOrder(
				runner.EXPECT().Output(gomock.Any(), workDir, gomock.Any(), psqlCommand,
					[]string{targetURL, "-v", "ON_ERROR_STOP=1", "-c", "REINDEX DATABASE template1;"}).
					Return(nil, nil),
				runner.EXPECT().Output(gomock.Any(), workDir, gomock.Any(), psqlCommand,
					[]string{targetURL, "-v", "ON_ERROR_STOP=1", "-c", "ALTER DATABASE template1 REFRESH COLLATION VERSION;"}).
					Return(nil, nil),
			)

			loadDSN := func() (string, string, error) {
				return "postgres://user@localhost:5432/local?sslmode=disable", "pw", nil
			}
			err := RunFix(context.Background(), runner, logging.NewTestLogger(t), "template1", loadDSN)
			require.NoError(t, err)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("不正なDB名はDSN解決前に弾く", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runner := mock_exec.NewMockRunner(ctrl)

			called := false
			loadDSN := func() (string, string, error) {
				called = true
				return "", "", nil
			}
			err := RunFix(context.Background(), runner, logging.NewTestLogger(t), "production", loadDSN)
			require.Error(t, err)
			assert.False(t, called)
		})

		t.Run("DSN解決に失敗すると修正せずエラー", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			runner := mock_exec.NewMockRunner(ctrl)

			loadDSN := func() (string, string, error) { return "", "", xerrors.New("config failed") }
			err := RunFix(context.Background(), runner, logging.NewTestLogger(t), "local", loadDSN)
			require.Error(t, err)
		})
	})
}
