package category

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

		t.Run("有効な値の場合、Categoryエンティティが生成される", func(t *testing.T) {
			t.Parallel()

			expected := &Category{
				id:      expectedUUID,
				name:    expectedName,
				code:    expectedCode,
				sortKey: expectedSortKey,
			}

			actual, err := New(expectedUUID, Attributes{Name: expectedName, Code: expectedCode, SortKey: expectedSortKey})
			require.NoError(t, err)
			assert.Equal(t, expected.id, actual.id)
			assert.Equal(t, expected.name, actual.name)
			assert.Equal(t, expected.code, actual.code)
			assert.Equal(t, expected.sortKey, actual.sortKey)
		})

		t.Run("コードが最小値ちょうど(1)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, Attributes{Name: expectedName, Code: minCode, SortKey: expectedSortKey})
			require.NoError(t, err)
			assert.Equal(t, minCode, actual.code)
		})

		t.Run("コードが最大値ちょうど(32767)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, Attributes{Name: expectedName, Code: maxCode, SortKey: expectedSortKey})
			require.NoError(t, err)
			assert.Equal(t, maxCode, actual.code)
		})

		t.Run("表示順が最小値ちょうど(1)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, Attributes{Name: expectedName, Code: expectedCode, SortKey: minSortKey})
			require.NoError(t, err)
			assert.Equal(t, minSortKey, actual.sortKey)
		})

		t.Run("表示順が最大値ちょうど(32767)の場合、生成される", func(t *testing.T) {
			t.Parallel()
			actual, err := New(expectedUUID, Attributes{Name: expectedName, Code: expectedCode, SortKey: maxSortKey})
			require.NoError(t, err)
			assert.Equal(t, maxSortKey, actual.sortKey)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDがゼロ値の場合、ErrInvalidIDエラーを返す", func(t *testing.T) {
			t.Parallel()

			res, err := New(uuid.UUID{}, Attributes{Name: expectedName, Code: expectedCode, SortKey: expectedSortKey})
			require.Nil(t, res)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("無効な商品カテゴリ名の場合、ErrInvalidNameエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("文字数が最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidName := strings.Repeat("字", minNameLength-1)
				res, err := New(expectedUUID, Attributes{Name: invalidName, Code: expectedCode, SortKey: expectedSortKey})
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidName)
			})

			t.Run("文字数が最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidName := strings.Repeat("字", maxNameLength+1)
				res, err := New(expectedUUID, Attributes{Name: invalidName, Code: expectedCode, SortKey: expectedSortKey})
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidName)
			})
		})

		t.Run("無効なコードの場合、ErrInvalidCodeエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("コードが最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidCode := minCode - 1
				res, err := New(expectedUUID, Attributes{Name: expectedName, Code: invalidCode, SortKey: expectedSortKey})
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidCode)
			})

			t.Run("コードが最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidCode := maxCode + 1
				res, err := New(expectedUUID, Attributes{Name: expectedName, Code: invalidCode, SortKey: expectedSortKey})
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidCode)
			})
		})

		t.Run("無効な表示順の場合、ErrInvalidSortKeyエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("表示順が最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidSortKey := minSortKey - 1
				res, err := New(expectedUUID, Attributes{Name: expectedName, Code: expectedCode, SortKey: invalidSortKey})
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidSortKey)
			})

			t.Run("表示順が最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidSortKey := maxSortKey + 1
				res, err := New(expectedUUID, Attributes{Name: expectedName, Code: expectedCode, SortKey: invalidSortKey})
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidSortKey)
			})
		})
	})
}

func TestCategory_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDメソッドは正しいIDを返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, Attributes{Name: "書籍", Code: 2, SortKey: 2})
			require.NoError(t, err)

			assert.Equal(t, id, category.ID())
		})
	})
}

func TestCategory_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Nameメソッドは正しい名前を返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, Attributes{Name: "書籍", Code: 2, SortKey: 2})
			require.NoError(t, err)

			assert.Equal(t, "書籍", category.Name())
		})
	})
}

func TestCategory_Code(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("Codeメソッドは正しいコードを返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, Attributes{Name: "書籍", Code: 2, SortKey: 2})
			require.NoError(t, err)

			assert.Equal(t, 2, category.Code())
		})
	})
}

func TestCategory_SortKey(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("SortKeyメソッドは正しい表示順を返す", func(t *testing.T) {
			t.Parallel()

			id, err := uuid.New()
			require.NoError(t, err)

			category, err := New(id, Attributes{Name: "書籍", Code: 2, SortKey: 3})
			require.NoError(t, err)

			assert.Equal(t, 3, category.SortKey())
		})
	})
}
