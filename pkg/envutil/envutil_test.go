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

		restoreExisting := Override(existingKey, "changed")
		v, ok := os.LookupEnv(existingKey)
		require.True(t, ok)
		assert.Equal(t, "changed", v)
		restoreExisting()
		v, ok = os.LookupEnv(existingKey)
		require.True(t, ok)
		assert.Equal(t, "original", v)

		restoreAbsent := Override(absentKey, "temp")
		v, ok = os.LookupEnv(absentKey)
		require.True(t, ok)
		assert.Equal(t, "temp", v)
		restoreAbsent()
		_, ok = os.LookupEnv(absentKey)
		assert.False(t, ok)
	})
}
