package ops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOpsPath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("/metricsはtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsOpsPath("/metrics"))
		})

		t.Run("/healthはtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsOpsPath("/health"))
		})

		t.Run("/healthzはtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsOpsPath("/healthz"))
		})

		t.Run("/readyはtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsOpsPath("/ready"))
		})

		t.Run("/versionはtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsOpsPath("/version"))
		})

		t.Run("末尾スラッシュ付きの運用系パスでもtrueを返す", func(t *testing.T) {
			t.Parallel()
			assert.True(t, IsOpsPath("/metrics/"))
		})

		t.Run("非運用系パスはfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsOpsPath("/non-ops"))
		})

		t.Run("ルートパスはfalseを返す", func(t *testing.T) {
			t.Parallel()
			assert.False(t, IsOpsPath("/"))
		})
	})
}
