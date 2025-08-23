package paging

import (
	"testing"

	"boilerplate-go/internal/apperror"

	"github.com/stretchr/testify/require"
)

func TestNewPageFrom1Based(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページ番号と件数がnilの場合、デフォルト値が使用される", func(t *testing.T) {
			actual, err := NewPageFrom1Based(nil, nil)
			expected := Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("ページ番号が1未満の場合、1として扱われる", func(t *testing.T) {
			page := -1
			actual, err := NewPageFrom1Based(&page, nil)
			expected := Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("件数が0以下の場合、デフォルト値が使用される", func(t *testing.T) {
			perPage := 0
			actual, err := NewPageFrom1Based(nil, &perPage)
			expected := Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("件数が最大値を超える場合、最大値が使用される", func(t *testing.T) {
			perPage := maxPerPage + 1
			actual, err := NewPageFrom1Based(nil, &perPage)
			expected := Paging{
				limit:  maxPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("ページ番号が3ページで1ページあたりの件数が50の場合、オフセットは100になる", func(t *testing.T) {
			page := 3
			perPage := 50
			expectedPageCount := 100
			actual, err := NewPageFrom1Based(&page, &perPage)
			expected := Paging{
				limit:  perPage,
				offset: expectedPageCount,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("ページ番号が最大値の場合、正しいオフセットが計算される", func(t *testing.T) {
			page := maxPage
			perPage := 10
			expectedOffset := (maxPage - 1) * perPage
			actual, err := NewPageFrom1Based(&page, &perPage)
			expected := Paging{
				limit:  perPage,
				offset: expectedOffset,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("件数が最大値の場合、正しいリミットが設定される", func(t *testing.T) {
			page := 1
			perPage := maxPerPage
			actual, err := NewPageFrom1Based(&page, &perPage)
			expected := Paging{
				limit:  maxPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})

		t.Run("ページ番号が負の値で、件数が負の値の場合、デフォルト値が使用される", func(t *testing.T) {
			page := -5
			perPage := -10
			actual, err := NewPageFrom1Based(&page, &perPage)
			expected := Paging{
				limit:  defaultPerPage,
				offset: 0,
			}

			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("ページ番号が最大値を超える場合、エラーが返される", func(t *testing.T) {
			page := maxPage + 1
			actual, err := NewPageFrom1Based(&page, nil)

			require.ErrorIs(t, err, apperror.ErrInvalidArgument)
			require.Zero(t, actual)
		})
	})
}

func TestPage_Getters(t *testing.T) {
	t.Parallel()

	t.Run("Limitが正しい値を返す", func(t *testing.T) {
		page := Paging{
			limit:  100,
			offset: 200,
		}

		actual := page.Limit()
		expected := 100

		require.Equal(t, expected, actual)
	})

	t.Run("Offsetが正しい値を返す", func(t *testing.T) {
		page := Paging{
			limit:  100,
			offset: 200,
		}

		actual := page.Offset()
		expected := 200

		require.Equal(t, expected, actual)
	})
}
