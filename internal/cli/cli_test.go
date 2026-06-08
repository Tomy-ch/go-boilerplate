package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterCommands(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "app"}
	RegisterCommands(root)

	// 期待する全サブコマンドが root に登録されていること。
	want := []string{
		"serve",
		"migrate-up",
		"migrate-down",
		"db-seed",
		"fix-collation",
		"dump-schema",
		"merge-dml",
		"job",
	}

	got := make(map[string]bool)
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}

	require.Len(t, root.Commands(), len(want))
	for _, name := range want {
		assert.True(t, got[name], "サブコマンド %q が登録されていること", name)
	}
}
