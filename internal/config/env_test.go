package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocalClassEnv(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("local 系（local/ci/test）は true を返す", func(t *testing.T) {
			t.Parallel()

			assert.True(t, IsLocalClassEnv(EnvLocal))
			assert.True(t, IsLocalClassEnv(EnvCI))
			assert.True(t, IsLocalClassEnv(EnvTest))
		})

		t.Run("deploy 系（dev/stg/prd）は false を返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, IsLocalClassEnv(EnvDevelopment))
			assert.False(t, IsLocalClassEnv(EnvStaging))
			assert.False(t, IsLocalClassEnv(EnvProduction))
		})

		t.Run("未知ラベル・空文字は false を返す", func(t *testing.T) {
			t.Parallel()

			assert.False(t, IsLocalClassEnv("unknown"))
			assert.False(t, IsLocalClassEnv(""))
		})
	})
}
