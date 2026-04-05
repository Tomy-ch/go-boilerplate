package module

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestControllerModule(t *testing.T) {
	t.Parallel()

	// このモジュールがnilでないことを確認するだけのテスト
	require.NotNil(t, ControllerModule())
}
