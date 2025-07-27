package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChdirToProjectRoot(t *testing.T) {
	t.Setenv("ENV", "test")

	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Chdir 後に戻すための defer
	defer func() {
		chdirErr := os.Chdir(originalDir)
		require.NoError(t, chdirErr)
	}()

	m := &testing.M{}
	ChdirToProjectRoot(m)

	wd, err := os.Getwd()
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(wd, "go.mod"))
	require.NoError(t, err)
}
