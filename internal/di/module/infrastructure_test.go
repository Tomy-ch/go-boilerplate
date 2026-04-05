package module

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInfrastructureModule(t *testing.T) {
	t.Parallel()

	// このモジュールがnilでないことを確認するだけのテスト
	require.NotNil(t, InfrastructureModule())
}
