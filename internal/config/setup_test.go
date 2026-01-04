package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetUpConfig_Succeeds(t *testing.T) {
	t.Run("正常に設定が読み込まれることを確認する", func(t *testing.T) {
		EnsureRepoRootAndEnv(t, TestingEnvValue)

		cfg, err := SetUpConfig()
		require.NoError(t, err)
		require.NotNil(t, cfg)
	})

	t.Run("設定の読み込みに失敗した場合、エラーが返されることを確認する", func(t *testing.T) {
		tmp := t.TempDir()
		t.Chdir(tmp)
		t.Setenv(envKey, "")

		cfg, err := SetUpConfig()
		require.Error(t, err)
		require.Nil(t, cfg)
	})
}
