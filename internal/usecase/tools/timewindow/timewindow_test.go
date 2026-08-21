package timewindow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-boilerplate/internal/apperror"
)

func TestNew(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	later := base.Add(24 * time.Hour)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("両方の境界を指定した場合、その半開区間になる", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{After: &base, Before: &later})
			require.NoError(t, err)
			assert.Equal(t, base, *got.After())
			assert.Equal(t, later, *got.Before())
		})

		t.Run("下限だけを指定した場合、上限に制限を設けない", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{After: &base})
			require.NoError(t, err)
			assert.Equal(t, base, *got.After())
			assert.Nil(t, got.Before())
		})

		t.Run("上限だけを指定した場合、下限に制限を設けない", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{Before: &later})
			require.NoError(t, err)
			assert.Nil(t, got.After())
			assert.Equal(t, later, *got.Before())
		})

		t.Run("両方を省略した場合、全期間を表す", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{})
			require.NoError(t, err)
			assert.Nil(t, got.After())
			assert.Nil(t, got.Before())
			assert.Equal(t, Window{}, got)
		})

		t.Run("上限が下限より1ナノ秒でも後なら通る", func(t *testing.T) {
			t.Parallel()
			justAfter := base.Add(time.Nanosecond)
			got, err := New(Bounds{After: &base, Before: &justAfter})
			require.NoError(t, err)
			assert.Equal(t, justAfter, *got.Before())
		})

		t.Run("生成後に呼び出し元が境界を書き換えても、対象期間は変わらない", func(t *testing.T) {
			t.Parallel()
			mutable := base
			got, err := New(Bounds{After: &mutable})
			require.NoError(t, err)
			mutable = later
			assert.Equal(t, base, *got.After())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限が下限より前の場合、不正引数エラーを返す", func(t *testing.T) {
			t.Parallel()
			_, err := New(Bounds{After: &later, Before: &base})
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})

		t.Run("上限と下限が同値の場合、空区間となるため不正引数エラーを返す", func(t *testing.T) {
			t.Parallel()
			same := base
			_, err := New(Bounds{After: &base, Before: &same})
			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
		})
	})
}

func TestWindow_After(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("下限を持つ場合、その瞬時を返す", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{After: &base})
			require.NoError(t, err)
			assert.Equal(t, base, *got.After())
		})

		t.Run("下限を持たない場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, Window{}.After())
		})

		t.Run("返り値を書き換えても、対象期間は変わらない", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{After: &base})
			require.NoError(t, err)
			returned := got.After()
			*returned = returned.Add(time.Hour)
			assert.Equal(t, base, *got.After())
		})
	})
}

func TestWindow_Before(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("上限を持つ場合、その瞬時を返す", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{Before: &base})
			require.NoError(t, err)
			assert.Equal(t, base, *got.Before())
		})

		t.Run("上限を持たない場合、nilを返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, Window{}.Before())
		})

		t.Run("返り値を書き換えても、対象期間は変わらない", func(t *testing.T) {
			t.Parallel()
			got, err := New(Bounds{Before: &base})
			require.NoError(t, err)
			returned := got.Before()
			*returned = returned.Add(time.Hour)
			assert.Equal(t, base, *got.Before())
		})
	})
}

func Test_copyTime(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("nilはnilのまま返す", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, copyTime(nil))
		})

		t.Run("同じ時刻を指す別のポインタを返す", func(t *testing.T) {
			t.Parallel()

			original := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			copied := copyTime(&original)

			require.NotNil(t, copied)
			assert.Equal(t, original, *copied)
			assert.NotSame(t, &original, copied)
		})
	})
}
