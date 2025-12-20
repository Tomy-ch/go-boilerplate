package datetime

import (
	"testing"
	"time"

	"boilerplate-go/internal/config"

	"github.com/stretchr/testify/require"
)

// NewTestLocation は、テスト用のタイムゾーンロケーションを生成します。
func NewTestLocation(t *testing.T) *time.Location {
	t.Helper()
	cfg := config.MockConfigForTest(t)
	osCfg := config.NewOSConfig(cfg)

	loc, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)
	return loc
}
