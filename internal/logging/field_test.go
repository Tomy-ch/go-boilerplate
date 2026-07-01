package logging

import (
	"testing"
	"time"

	"go-boilerplate/pkg/xerrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("string値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedString := "string"
			expected := &Field{
				key:         expectedKey,
				kind:        fieldString,
				stringValue: expectedString,
			}

			assert.Equal(t, expected, String(expectedKey, expectedString))
		})
	})
}

func TestStrings(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("[]string値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedStrings := []string{"one", "two", "three"}
			expected := &Field{
				key:          expectedKey,
				kind:         fieldStrings,
				stringsValue: expectedStrings,
			}

			assert.Equal(t, expected, Strings(expectedKey, expectedStrings))
		})
	})
}

func TestInt(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedInt := 42
			expected := &Field{
				key:      expectedKey,
				kind:     fieldInt,
				intValue: expectedInt,
			}

			assert.Equal(t, expected, Int(expectedKey, expectedInt))
		})
	})
}

func TestInt64(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("int64値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			var expectedInt64 int64 = 4200000000
			expected := &Field{
				key:        expectedKey,
				kind:       fieldInt64,
				int64Value: expectedInt64,
			}

			assert.Equal(t, expected, Int64(expectedKey, expectedInt64))
		})
	})
}

func TestFloat64(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("float64値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedFloat64 := 3.14159
			expected := &Field{
				key:          expectedKey,
				kind:         fieldFloat64,
				float64Value: expectedFloat64,
			}

			assert.Equal(t, expected, Float64(expectedKey, expectedFloat64))
		})
	})
}

func TestBool(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("bool値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedBool := true
			expected := &Field{
				key:       expectedKey,
				kind:      fieldBool,
				boolValue: expectedBool,
			}

			assert.Equal(t, expected, Bool(expectedKey, expectedBool))
		})
	})
}

func TestTime(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("RFC3339Nano文字列に変換したFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedTime := time.Date(2024, time.January, 2, 15, 4, 5, 0, time.UTC)
			expected := &Field{
				key:         expectedKey,
				kind:        fieldString,
				stringValue: expectedTime.Format(time.RFC3339Nano),
			}

			assert.Equal(t, expected, Time(expectedKey, expectedTime))
		})
	})
}

func TestDurationMs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ミリ秒のfloat64に変換したFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedDuration := 1500 * time.Millisecond
			expected := &Field{
				key:          expectedKey,
				kind:         fieldFloat64,
				float64Value: 1500.0,
			}

			assert.Equal(t, expected, DurationMs(expectedKey, expectedDuration))
		})
	})
}

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("error値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedError := xerrors.New("something went wrong")
			expected := &Field{
				key:        expectedKey,
				kind:       fieldError,
				errorValue: expectedError,
			}

			assert.Equal(t, expected, Error(expectedKey, expectedError))
		})

		t.Run("nilエラーの場合、errorValueがnilのFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expected := &Field{
				key:        expectedKey,
				kind:       fieldError,
				errorValue: nil,
			}

			assert.Equal(t, expected, Error(expectedKey, nil))
		})
	})
}

func TestStacktrace(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("StackTraceを行配列として保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedError := xerrors.New("something went wrong")
			expected := &Field{
				key:          expectedKey,
				kind:         fieldStrings,
				stringsValue: SplitStackLines(xerrors.StackTrace(expectedError)),
			}

			assert.Equal(t, expected, Stacktrace(expectedKey, expectedError))
		})

		t.Run("nilエラーの場合、空のFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expected := &Field{
				key:          expectedKey,
				kind:         fieldStrings,
				stringsValue: nil,
			}

			assert.Equal(t, expected, Stacktrace(expectedKey, nil))
		})
	})
}

func TestSplitStackLines(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("複数行を改行で分割し各行の先頭タブを除いた行配列を返す", func(t *testing.T) {
			t.Parallel()

			input := "frame1\n\t/path/to/file.go:10\nframe2\n\t/path/to/other.go:20"
			expected := []string{
				"frame1",
				"/path/to/file.go:10",
				"frame2",
				"/path/to/other.go:20",
			}
			assert.Equal(t, expected, SplitStackLines(input))
		})

		t.Run("先頭タブが複数連続していても全て除去される", func(t *testing.T) {
			t.Parallel()

			input := "frame1\n\t\t/path/to/file.go:10"
			expected := []string{"frame1", "/path/to/file.go:10"}
			assert.Equal(t, expected, SplitStackLines(input))
		})

		t.Run("cockroachdb_errorsのガター（空白・パイプ・タブ）を行頭から除去する", func(t *testing.T) {
			t.Parallel()

			input := "  | go-boilerplate/pkg/xerrors.Join\n  | \t/app/pkg/xerrors/errors.go:34\n  -- stack trace:"
			expected := []string{
				"go-boilerplate/pkg/xerrors.Join",
				"/app/pkg/xerrors/errors.go:34",
				"-- stack trace:",
			}
			assert.Equal(t, expected, SplitStackLines(input))
		})

		t.Run("末尾改行は分割結果から除去される", func(t *testing.T) {
			t.Parallel()

			input := "frame1\nframe2\n"
			expected := []string{"frame1", "frame2"}
			assert.Equal(t, expected, SplitStackLines(input))
		})

		t.Run("空文字列の場合、nilを返す", func(t *testing.T) {
			t.Parallel()

			assert.Nil(t, SplitStackLines(""))
		})
	})
}

func TestAny(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("任意型値を保持するFieldを生成する", func(t *testing.T) {
			t.Parallel()

			const expectedKey = "key"
			expectedAny := map[string]int{"one": 1, "two": 2}
			expected := &Field{
				key:      expectedKey,
				kind:     fieldAny,
				anyValue: expectedAny,
			}

			assert.Equal(t, expected, Any(expectedKey, expectedAny))
		})
	})
}

func Test_latencyMs(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ミリ秒変換が正しい", func(t *testing.T) {
			t.Parallel()
			ms := latencyMs(250 * time.Millisecond)
			require.InEpsilon(t, float64(250), ms, 0.01)
		})
	})
}
