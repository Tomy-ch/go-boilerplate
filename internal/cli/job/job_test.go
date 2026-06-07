package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommand(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	require.NotNil(t, cmd)
	assert.Equal(t, "job", cmd.Use)

	// timeout フラグが既定値 0 で登録されていること。
	f := cmd.Flags().Lookup("timeout")
	require.NotNil(t, f)
	assert.Equal(t, "0s", f.DefValue)

	// 引数なしの実行は MinimumNArgs(1) により拒否されること。
	require.Error(t, cmd.Args(cmd, []string{}))
	require.NoError(t, cmd.Args(cmd, []string{"usercount"}))
}
