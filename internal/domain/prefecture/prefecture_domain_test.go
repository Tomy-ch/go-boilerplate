package prefecture

import (
	"strings"
	"testing"

	"boilerplate-go/pkg/uuid"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	expectedUUID, err := uuid.New()
	require.NoError(t, err)
	expectedName := "Tokyo"
	expectedCode := 13

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な都道府県名の場合、Prefectureエンティティが生成される", func(t *testing.T) {
			t.Parallel()

			expected := &Entity{
				id:   expectedUUID,
				name: expectedName,
				code: expectedCode,
			}

			actual, err := New(expectedUUID.String(), expectedName, expectedCode)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("無効なIDの場合、ErrInvalidIDエラーが返される", func(t *testing.T) {
			t.Parallel()

			res, err := New("invalid-uuid", "Tokyo", expectedCode)
			require.Nil(t, res)
			require.ErrorIs(t, err, ErrInvalidID)
		})

		t.Run("無効な都道府県名の場合、ErrInvalidPrefectureNameエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("文字数が最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidName := strings.Repeat("字", MinPrefectureNameLength-1)
				res, err := New(expectedUUID.String(), invalidName, expectedCode)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidPrefectureName)
			})

			t.Run("文字数が最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidName := strings.Repeat("字", MaxPrefectureNameLength+1)
				res, err := New(expectedUUID.String(), invalidName, expectedCode)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidPrefectureName)
			})
		})

		t.Run("無効なコードの場合、ErrInvalidCodeエラーが返される", func(t *testing.T) {
			t.Parallel()

			t.Run("コードが最小値未満の場合", func(t *testing.T) {
				t.Parallel()

				invalidCode := MinCode - 1
				res, err := New(expectedUUID.String(), expectedName, invalidCode)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidCode)
			})

			t.Run("コードが最大値超過の場合", func(t *testing.T) {
				t.Parallel()

				invalidCode := MaxCode + 1
				res, err := New(expectedUUID.String(), expectedName, invalidCode)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidCode)
			})
		})
	})
}

func TestEntity_Accessors(t *testing.T) {
	t.Parallel()

	id, err := uuid.New()
	require.NoError(t, err)
	name := "Osaka"
	code := 27

	prefecture, err := New(id.String(), name, code)
	require.NoError(t, err)

	t.Run("IDメソッドは正しいIDを返す", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, id, prefecture.ID())
	})

	t.Run("Nameメソッドは正しい名前を返す", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, name, prefecture.Name())
	})

	t.Run("Codeメソッドは正しいコードを返す", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, code, prefecture.Code())
	})
}
