package config

import (
	"bytes"
	"os"
	"strings"
	"testing"

	root "go-boilerplate"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // Load は os.Setenv でグローバル環境を変更するため並列化不可
func TestLoad(t *testing.T) {
	restoreEnvAfterTest(t)

	t.Run("正常系", func(t *testing.T) {
		t.Run("埋め込み env を読み込み環境変数へ反映する", func(t *testing.T) {
			_ = os.Unsetenv("APP_NAME")

			require.NoError(t, Load())
			assert.Equal(t, "Boilerplate", os.Getenv("APP_NAME"))
		})

		t.Run("既存の環境変数は上書きしない", func(t *testing.T) {
			t.Setenv("APP_NAME", "existing-value")

			require.NoError(t, Load())
			assert.Equal(t, "existing-value", os.Getenv("APP_NAME"))
		})

		t.Run("埋め込み env の APP_ENV 素性をパッケージ変数へ捕捉する", func(t *testing.T) {
			// OS 側で APP_ENV を上書きしても、捕捉されるのはあくまで埋め込み値であること。
			// 期待値を同じ埋め込み env から再導出して完全一致を検証するため materialize 対象の env に
			// 依存せず、別キーへの捕捉退行を検出できる（値が偶然 APP_ENV と一致するキーはローカルの
			// env/.env では区別できないが、CI が materialize する env/.env.ci では全キーを検出できる）。
			t.Setenv("APP_ENV", "injected-at-runtime")
			prev := embeddedAppEnv
			t.Cleanup(func() { embeddedAppEnv = prev })
			embeddedAppEnv = ""

			require.NoError(t, Load())
			b, err := root.FS.ReadFile("env/.env")
			require.NoError(t, err)
			embedded, err := godotenv.Parse(bytes.NewReader(b))
			require.NoError(t, err)
			assert.Equal(t, embedded["APP_ENV"], embeddedAppEnv)
		})
	})
}

// restoreEnvAfterTest は、テスト終了時に環境変数をテスト開始時点へ戻します。
func restoreEnvAfterTest(t *testing.T) {
	t.Helper()
	snapshot := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, kv := range snapshot {
			k, v, _ := strings.Cut(kv, "=")
			//nolint:usetesting // テスト後の環境変数一括復元のため t.Setenv は使用できない
			_ = os.Setenv(k, v)
		}
	})
}
