package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestLocation(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("MockConfig のタイムゾーンに対応するロケーションが返ること", func(t *testing.T) {
			t.Parallel()

			cfg := MockConfigForTest(t)
			osCfg := NewOperatingSystemConfig(cfg)

			expected, err := time.LoadLocation(osCfg.TimeZone())
			require.NoError(t, err)

			actual := NewTestLocation(t)
			assert.Equal(t, expected, actual)
		})
	})
}

func TestResolvedAuthIssuer(t *testing.T) {
	// t.Setenv でプロセス状態を書き換えるため Parallel は使用しない。
	t.Run("正常系", func(t *testing.T) {
		t.Run("実行時の環境変数があればその値を返す", func(t *testing.T) {
			t.Setenv(authIssuerEnvKey, "http://localhost:4007")

			assert.Equal(t, "http://localhost:4007", ResolvedAuthIssuer(t))
		})

		t.Run("環境変数が無ければ埋め込み env の値を返す", func(t *testing.T) {
			t.Setenv(authIssuerEnvKey, "")

			kv, err := godotenv.Read(filepath.Join(repoRoot(t), "env", ".env"))
			require.NoError(t, err)
			assert.Equal(t, kv[authIssuerEnvKey], ResolvedAuthIssuer(t))
		})
	})
}

func Test_repoRoot(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("go.mod を含むリポジトリルートを返す", func(t *testing.T) {
			t.Parallel()

			root := repoRoot(t)
			assert.FileExists(t, filepath.Join(root, "go.mod"))
		})
	})
}

func TestEnsureRepoRootAndEnv(t *testing.T) {
	// t.Setenv / t.Chdir でプロセス状態を書き換えるため Parallel は使用しない。
	orig, err := os.Getwd()
	require.NoError(t, err)

	t.Run("正常系", func(t *testing.T) {
		t.Run("リポジトリルートへ移動し対象 env の値を環境変数へ設定する", func(t *testing.T) { //nolint:paralleltest // t.Setenv/t.Chdir使用のため並列化不可
			EnsureRepoRootAndEnv(t, TestingEnvValue)

			cwd, inErr := os.Getwd()
			require.NoError(t, inErr)
			assert.FileExists(t, filepath.Join(cwd, "go.mod"))
			// env/.env.<env> の値が環境変数へ反映される。
			assert.Equal(t, TestingEnvValue, os.Getenv("APP_ENV"))
		})

		t.Run("ci 環境で DB_NAME_TEST が設定されていれば DB_NAME をそれへ上書きする", func(t *testing.T) {
			t.Setenv("DB_NAME_TEST", "wt9_test")

			EnsureRepoRootAndEnv(t, TestingEnvValue)

			assert.Equal(t, "wt9_test", os.Getenv("DB_NAME"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("deploy 系 env では DB_NAME_TEST を無視し .env.<env> の DB_NAME を維持する", func(t *testing.T) {
			t.Setenv("DB_NAME_TEST", "wt9_test")

			EnsureRepoRootAndEnv(t, EnvDevelopment)

			// env/.env.dev の DB_NAME がそのまま残り、wt9_test へは上書きされない。
			assert.NotEqual(t, "wt9_test", os.Getenv("DB_NAME"))
		})
	})

	// サブテスト終了時に t.Chdir がクリーンアップされ、cwd が復元されること。
	cwdAfter, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, orig, cwdAfter)
}

func Test_setWorktreeDBName(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("local 環境は DB_NAME_LOCAL へ上書きする", func(t *testing.T) {
			t.Setenv("DB_NAME_LOCAL", "wt9_local")

			setWorktreeDBName(t, EnvLocal)

			assert.Equal(t, "wt9_local", os.Getenv("DB_NAME"))
		})

		t.Run("ci 環境は DB_NAME_TEST へ上書きする", func(t *testing.T) {
			t.Setenv("APP_ENV", EnvCI)
			t.Setenv("DB_NAME_TEST", "wt9_test")

			setWorktreeDBName(t, EnvCI)

			assert.Equal(t, "wt9_test", os.Getenv("DB_NAME"))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("deploy 系 env は DB_NAME を上書きしない", func(t *testing.T) {
			t.Setenv("DB_NAME", "prd_db")
			t.Setenv("DB_NAME_TEST", "wt9_test")

			setWorktreeDBName(t, EnvProduction)

			assert.Equal(t, "prd_db", os.Getenv("DB_NAME"))
		})
	})
}

func Test_testDBName(t *testing.T) {
	t.Run("正常系", func(t *testing.T) {
		t.Run("ci 環境では DB_NAME_TEST を尊重する", func(t *testing.T) {
			t.Setenv("APP_ENV", EnvCI)
			t.Setenv("DB_NAME_TEST", "wt9_test")

			assert.Equal(t, "wt9_test", testDBName())
		})

		t.Run("APP_ENV 未設定でも DB_NAME_TEST を尊重する", func(t *testing.T) {
			t.Setenv("APP_ENV", "")
			t.Setenv("DB_NAME_TEST", "wt9_test")

			assert.Equal(t, "wt9_test", testDBName())
		})

		t.Run("DB_NAME_TEST 未設定なら既定 test を返す", func(t *testing.T) {
			t.Setenv("APP_ENV", EnvCI)
			t.Setenv("DB_NAME_TEST", "")

			assert.Equal(t, defaultTestDBName, testDBName())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("prd 環境では DB_NAME_TEST を無視して既定 test へフォールバックする", func(t *testing.T) {
			t.Setenv("APP_ENV", EnvProduction)
			t.Setenv("DB_NAME_TEST", "wt9_test")

			assert.Equal(t, defaultTestDBName, testDBName())
		})

		t.Run("stg 環境でも DB_NAME_TEST を無視する", func(t *testing.T) {
			t.Setenv("APP_ENV", EnvStaging)
			t.Setenv("DB_NAME_TEST", "wt9_test")

			assert.Equal(t, defaultTestDBName, testDBName())
		})
	})
}
