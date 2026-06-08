package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTimeLocation(t *testing.T) {
	t.Parallel()

	t.Run("有効なタイムゾーンの場合、正しいロケーションが返ること", func(t *testing.T) {
		t.Parallel()
		cfg := MockConfigForTest(t)
		osCfg := NewOperationSystemConfig(cfg)

		loc, err := NewTimeLocation(osCfg)
		require.NoError(t, err)
		assert.Equal(t, expectedOSTimeZone, loc.String())
	})

	t.Run("無効なタイムゾーンの場合、エラーが返ること", func(t *testing.T) {
		t.Parallel()

		invalidCfg := MockConfigForTest(t)
		invalidCfg.os.timezone = "Invalid/Timezone"
		osCfg := NewOperationSystemConfig(invalidCfg)

		loc, err := NewTimeLocation(osCfg)
		require.Error(t, err)
		require.Nil(t, loc)
	})
}
