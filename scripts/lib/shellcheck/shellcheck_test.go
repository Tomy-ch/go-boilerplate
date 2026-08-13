package shellcheck_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/scripts/lib/shellcheck"
)

const requireShellcheckEnv = "REQUIRE_SHELLCHECK"

const cleanScript = "#!/bin/sh\nset -eu\necho hello\n"

// SC2086: 引用符の無い展開。方言に依存せず必ず指摘される。
const dirtyScript = "#!/bin/sh\nset -eu\nx=$1\necho $x\n"

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("指摘の無い内容には何も返さない", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			out, err := shellcheck.Run(t.Context(), cleanScript)

			require.NoError(t, err)
			assert.Empty(t, strings.TrimSpace(out))
		})

		t.Run("指摘のある内容を gcc 形式で返す", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			out, err := shellcheck.Run(t.Context(), dirtyScript)

			require.NoError(t, err)
			assert.Contains(t, out, "SC2086")
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("起動できなければ指摘と区別して報告する", func(t *testing.T) {
			t.Parallel()
			requireShellcheck(t)

			_, err := shellcheck.Run(canceledContext(t), cleanScript)

			require.ErrorIs(t, err, shellcheck.ErrRun)
		})
	})
}

func requireShellcheck(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath(shellcheck.Binary); err == nil {
		return
	}
	if os.Getenv(requireShellcheckEnv) != "" {
		t.Fatalf("shellcheck が PATH にありません（%s 指定時は skip しません）", requireShellcheckEnv)
	}
	t.Skip("shellcheck が PATH にありません")
}

func canceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	return ctx
}
