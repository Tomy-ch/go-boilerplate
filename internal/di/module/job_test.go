package module

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJobModule(t *testing.T) {
	t.Parallel()

	// このモジュールがnilでないことを確認するだけのテスト
	require.NotNil(t, JobModule())
}

func Test_provideJobs(t *testing.T) {
	t.Parallel()

	// この関数がnilでないことを確認するだけのテスト
	require.NotNil(t, provideJobs())
}
