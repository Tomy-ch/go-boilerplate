package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBuildInfo(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ldflags未注入のデフォルト値を保持したBuildInfoを返す", func(t *testing.T) {
			t.Parallel()

			expected := &buildInfo{
				version:   Version,
				revision:  Revision,
				buildDate: BuildDate,
			}

			assert.Equal(t, expected, NewBuildInfo())
		})
	})
}

// newTestBuildInfo は、3 項目を互いに区別できる値で埋めた buildInfo を返します。
func newTestBuildInfo() *buildInfo {
	return &buildInfo{
		version:   "1.0.0",
		revision:  "abc123",
		buildDate: "2024-12-31T21:00:00Z",
	}
}

func Test_buildInfo_BuildDate(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時に渡したビルド日時を返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "2024-12-31T21:00:00Z", newTestBuildInfo().BuildDate())
		})
	})
}

func Test_buildInfo_Revision(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時に渡したリビジョンを返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "abc123", newTestBuildInfo().Revision())
		})
	})
}

func Test_buildInfo_Version(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時に渡したバージョンを返す", func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, "1.0.0", newTestBuildInfo().Version())
		})
	})
}
