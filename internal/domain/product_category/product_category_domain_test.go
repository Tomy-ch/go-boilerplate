package productcategory

import (
	"strings"
	"testing"

	"go-boilerplate/pkg/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	expectedUUID, err := uuid.New()
	require.NoError(t, err)
	expectedName := "電子機器"
	expectedCode := 1
	expectedSortKey := 1

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な値の場合、ProductCategoryエンティティが生成される", func(t *testing.T) {
			t.Parallel()

			expected := &ProductCategory{
				id:      expectedUUID,
				name:    expectedName,
				code:    expectedCode,
				sortKey: expectedSortKey,
			}

			actual, err := New(expectedUUID, expectedName, expectedCode, expectedSortKey)
			require.NoError(t, err)
			assert.Equal(t, expected.id, actual.id)
			assert.Equal(t, expected.name, actual.name)
			assert.Equal(t, expected.code, actual.code)
			assert.Equal(t, expected.sortKey, actual.sortKey)
		})

		t.Run("コードが最小値ちょうど(1)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, expectedName, minCode, expectedSortKey)
			require.NoError(t, err)
			assert.Equal(t, minCode, actual.code)
		})

		t.Run("コードが最大値ちょうど(32767)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, expectedName, maxCode, expectedSortKey)
			require.NoError(t, err)
			assert.Equal(t, maxCode, actual.code)
		})

		t.Run("表示順が最小値ちょうど(1)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, expectedName, expectedCode, minSortKey)
			require.NoError(t, err)
			assert.Equal(t, minSortKey, actual.sortKey)
		})

		t.Run("表示順が最大値ちょうど(32767)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, expectedName, expectedCode, maxSortKey)
			require.NoError(t, err)
			assert.Equal(t, maxSortKey, actual.sortKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDエラーを返す", func(t *testing.T) {
			t.Parallel()

			res, err := New(uuid.UUID{}, expectedName, expectedCode, expectedSortKey)
			require.Nil(t, res)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("無効な商品カテゴリ名の場合、ErrInvalidNameエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("文字数が最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidName := strings.Repeat("字", minNameLength-1)
				res, err := New(expectedUUID, invalidName, expectedCode, expectedSortKey)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidName)
			})

			t.Run("文字数が最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidName := strings.Repeat("字", maxNameLength+1)
				res, err := New(expectedUUID, invalidName, expectedCode, expectedSortKey)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidName)
			})
		})

		t.Run("無効なコードの場合、ErrInvalidCodeエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("コードが最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidCode := minCode - 1
				res, err := New(expectedUUID, expectedName, invalidCode, expectedSortKey)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidCode)
			})

			t.Run("コードが最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidCode := maxCode + 1
				res, err := New(expectedUUID, expectedName, invalidCode, expectedSortKey)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidCode)
			})
		})

		t.Run("無効な表示順の場合、ErrInvalidSortKeyエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("表示順が最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidSortKey := minSortKey - 1
				res, err := New(expectedUUID, expectedName, expectedCode, invalidSortKey)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidSortKey)
			})

			t.Run("表示順が最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidSortKey := maxSortKey + 1
				res, err := New(expectedUUID, expectedName, expectedCode, invalidSortKey)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidSortKey)
			})
		})
	})
}

func TestProductCategory_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDメソッドは正しいIDを返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, "書籍", 2, 2)
			require.NoError(t, err)

			assert.Equal(t, id, category.ID())
		})
	})
}

func TestProductCategory_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Nameメソッドは正しい名前を返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, "書籍", 2, 2)
			require.NoError(t, err)

			assert.Equal(t, "書籍", category.Name())
		})
	})
}

func TestProductCategory_Code(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Codeメソッドは正しいコードを返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, "書籍", 2, 2)
			require.NoError(t, err)

			assert.Equal(t, 2, category.Code())
		})
	})
}

func TestProductCategory_SortKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SortKeyメソッドは正しい表示順を返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, "書籍", 2, 3)
			require.NoError(t, err)

			assert.Equal(t, 3, category.SortKey())
		})
	})
}
