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

func Test_buildInfo_Methods(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Version/Revision/BuildDateはコンストラクタで渡した値を返す", func(t *testing.T) {
			t.Parallel()

			expectedVersion := "1.0.0"
			expectedRevision := "abc123"
			expectedBuildDate := "2024-12-31T21:00:00Z"

			actual := &buildInfo{
				version:   expectedVersion,
				revision:  expectedRevision,
				buildDate: expectedBuildDate,
			}

			assert.Equal(t, expectedVersion, actual.Version())
			assert.Equal(t, expectedRevision, actual.Revision())
			assert.Equal(t, expectedBuildDate, actual.BuildDate())
		})
	})
}

func Test_buildInfo_BuildDate(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func Test_buildInfo_Revision(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}

func Test_buildInfo_Version(t *testing.T) {
	t.Parallel()
	t.Skip("architest の 1:1 検証を全 func / method へ拡張した際の宣言。実テストは #724 で追加する")
}
