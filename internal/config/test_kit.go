package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// NewTestLocation は、テスト用のタイムゾーンロケーションを生成します。
func NewTestLocation(t *testing.T) *time.Location {
	t.Helper()
	cfg := MockConfigForTest(t)
	osCfg := NewOperationSystemConfig(cfg)

	loc, err := time.LoadLocation(osCfg.TimeZone())
	require.NoError(t, err)
	return loc
}

// EnsureRepoRootAndEnv moves working directory to repository root (where go.mod exists)
// if found, registers a cleanup to restore the original working directory, and
// sets the `ENV` environment variable for the test using `t.Setenv`.
func EnsureRepoRootAndEnv(t *testing.T, env string) {
	t.Helper()

	orig, err := os.Getwd()
	require.NoError(t, err)
	// Ensure we restore the original working directory even if `t.Chdir` is
	// not available (older Go) or if an early return occurs.
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})

	// find repo root by locating go.mod upwards
	p := orig
	for {
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			t.Chdir(p)
			break
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}

	t.Setenv(envKey, env)
}
