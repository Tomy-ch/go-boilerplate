package testUtil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChdirToProjectRoot(t *testing.T) {
	t.Setenv("ENV", "test")

	originalDir, err := os.Getwd()
	assert.NoError(t, err)

	// Chdir 後に戻すための defer
	defer func() {
		err := os.Chdir(originalDir)
		assert.NoError(t, err)
	}()

	m := &testing.M{}
	ChdirToProjectRoot(m)

	wd, err := os.Getwd()
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(wd, "go.mod"))
	assert.NoError(t, err)
}
