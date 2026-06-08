package envutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverride(t *testing.T) {
	// 環境変数を触るため、この関数は t.Parallel を使わない。

	t.Run("正常系_既存値は復元され未設定値はUnsetされる", func(t *testing.T) {
		const existingKey = "TEST_ENVUTIL_EXISTING"
		const absentKey = "TEST_ENVUTIL_ABSENT"

		t.Setenv(existingKey, "original")

		restoreExisting, err := Override(existingKey, "changed")
		require.NoError(t, err)
		v, ok := os.LookupEnv(existingKey)
		require.True(t, ok)
		assert.Equal(t, "changed", v)
		restoreExisting()
		v, ok = os.LookupEnv(existingKey)
		require.True(t, ok)
		assert.Equal(t, "original", v)

		restoreAbsent, err := Override(absentKey, "temp")
		require.NoError(t, err)
		v, ok = os.LookupEnv(absentKey)
		require.True(t, ok)
		assert.Equal(t, "temp", v)
		restoreAbsent()
		_, ok = os.LookupEnv(absentKey)
		assert.False(t, ok)
	})

	t.Run("異常系_不正なキーはエラーを返し副作用を残さない", func(t *testing.T) {
		// キーに '=' を含むと os.Setenv は失敗する。
		restore, err := Override("BAD=KEY", "x")
		require.Error(t, err)
		require.NotNil(t, restore) // 復元関数は no-op でも非 nil を返す
		_, ok := os.LookupEnv("BAD=KEY")
		assert.False(t, ok)
	})
}
