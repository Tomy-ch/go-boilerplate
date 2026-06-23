package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // SetUpConfig は os.Setenv でグローバル環境を変更するため並列化不可
func TestSetUpConfig_Succeeds(t *testing.T) {
	restoreEnvAfterTest(t)

	t.Run("正常系", func(t *testing.T) {
		t.Run("正常に設定が読み込まれることを確認する", func(t *testing.T) {
			cfg, err := SetUpConfig()
			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Run("設定値が不正な場合、エラーが返されることを確認する", func(t *testing.T) {
			// Load は既存の環境変数を上書きしないため、不正な APP_MODE が残り検証で落ちる。
			t.Setenv("APP_MODE", "invalid-mode")

			cfg, err := SetUpConfig()
			require.Error(t, err)
			require.Nil(t, cfg)
		})
	})
}
