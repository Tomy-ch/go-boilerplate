package paging

import (
	"testing"

	"go-boilerplate/internal/apperror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPageFrom1Based(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページ番号と件数がnilの場合、デフォルト値が使用される", func(t *testing.T) {
			t.Parallel()
			actual, err := NewPagingFrom1Based(nil, nil)
			expected := &Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("ページ番号が1未満の場合、1として扱われる", func(t *testing.T) {
			t.Parallel()
			page := -1
			actual, err := NewPagingFrom1Based(&page, nil)
			expected := &Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("件数が0以下の場合、デフォルト値が使用される", func(t *testing.T) {
			t.Parallel()
			perPage := 0
			actual, err := NewPagingFrom1Based(nil, &perPage)
			expected := &Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("件数が最大値を超える場合、最大値が使用される", func(t *testing.T) {
			t.Parallel()
			perPage := maxPerPage + 1
			actual, err := NewPagingFrom1Based(nil, &perPage)
			expected := &Paging{
				limit:  maxPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("ページ番号が3ページで1ページあたりの件数が50の場合、オフセットは100になる", func(t *testing.T) {
			t.Parallel()
			page := 3
			perPage := 50
			expectedPageCount := 100
			actual, err := NewPagingFrom1Based(&page, &perPage)
			expected := &Paging{
				limit:  perPage,
				offset: expectedPageCount,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("ページ番号が最大値の場合、正しいオフセットが計算される", func(t *testing.T) {
			t.Parallel()
			page := maxPage
			perPage := 10
			expectedOffset := (maxPage - 1) * perPage
			actual, err := NewPagingFrom1Based(&page, &perPage)
			expected := &Paging{
				limit:  perPage,
				offset: expectedOffset,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("件数が最大値の場合、正しいリミットが設定される", func(t *testing.T) {
			t.Parallel()
			page := 1
			perPage := maxPerPage
			actual, err := NewPagingFrom1Based(&page, &perPage)
			expected := &Paging{
				limit:  maxPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})

		t.Run("ページ番号が負の値で、件数が負の値の場合、デフォルト値が使用される", func(t *testing.T) {
			t.Parallel()
			page := -5
			perPage := -10
			actual, err := NewPagingFrom1Based(&page, &perPage)
			expected := &Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			assert.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページ番号が最大値を超える場合、エラーが返される", func(t *testing.T) {
			t.Parallel()
			page := maxPage + 1
			actual, err := NewPagingFrom1Based(&page, nil)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Zero(t, actual)
		})
	})
}

func TestPage_Getters(t *testing.T) {
	t.Parallel()

	t.Run("Limitが正しい値を返す", func(t *testing.T) {
		t.Parallel()
		page := &Paging{
			limit:  100,
			offset: 200,
		}

		actual := page.Limit()
		expected := 100

		assert.Equal(t, expected, actual)
	})

	t.Run("Offsetが正しい値を返す", func(t *testing.T) {
		t.Parallel()
		page := &Paging{
			limit:  100,
			offset: 200,
		}

		actual := page.Offset()
		expected := 200

		assert.Equal(t, expected, actual)
	})

	t.Run("Limit32が正しい値を返す", func(t *testing.T) {
		t.Parallel()
		page := &Paging{
			limit:  100,
			offset: 200,
		}

		actual := page.Limit32()
		expected := int32(100)

		assert.Equal(t, expected, actual)
	})

	t.Run("Limit32がmaxPerPageを超える場合はクランプされる", func(t *testing.T) {
		t.Parallel()
		page := &Paging{
			limit:  maxPerPage + 100,
			offset: 0,
		}

		actual := page.Limit32()
		expected := int32(maxPerPage)

		assert.Equal(t, expected, actual)
	})

	t.Run("Offset32が正しい値を返す", func(t *testing.T) {
		t.Parallel()
		page := &Paging{
			limit:  100,
			offset: 200,
		}

		actual := page.Offset32()
		expected := int32(200)

		assert.Equal(t, expected, actual)
	})

	t.Run("Offset32が最大値を超える場合はクランプされる", func(t *testing.T) {
		t.Parallel()
		page := &Paging{
			limit:  0,
			offset: maxPage*maxPerPage + 1000,
		}

		actual := page.Offset32()
		expected := int32(maxPage * maxPerPage)

		assert.Equal(t, expected, actual)
	})
}
