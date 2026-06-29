package config

import (
	"os"
	"strings"
	"testing"

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
