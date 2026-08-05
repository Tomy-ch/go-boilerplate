package product

import (
	"strings"
	"testing"

	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusRef(t *testing.T) {
	t.Parallel()

	id := uuidtestkit.NewTestFromSalt(t, "status_ref_id")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な id と名称の場合、参照が生成され ID/Name を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewStatusRef(id, "在庫あり")
			require.NoError(t, err)
			assert.Equal(t, id, ref.ID())
			assert.Equal(t, "在庫あり", ref.Name())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("id がゼロ値の場合、ErrInvalidStatusID を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewStatusRef(uuid.UUID{}, "在庫あり")
			require.ErrorIs(t, err, ErrInvalidStatusID)
			assert.Equal(t, StatusRef{}, ref)
		})

		t.Run("名称が空の場合、ErrInvalidStatusName を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewStatusRef(id, "")
			require.ErrorIs(t, err, ErrInvalidStatusName)
			assert.Equal(t, StatusRef{}, ref)
		})

		t.Run("名称が最大長を超える場合、ErrInvalidStatusName を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewStatusRef(id, strings.Repeat("あ", maxRefNameLength+1))
			require.ErrorIs(t, err, ErrInvalidStatusName)
			assert.Equal(t, StatusRef{}, ref)
		})
	})
}

func TestNewCategoryRef(t *testing.T) {
	t.Parallel()

	id := uuidtestkit.NewTestFromSalt(t, "category_ref_id")

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("有効な id と名称の場合、参照が生成され ID/Name を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewCategoryRef(id, "電子機器")
			require.NoError(t, err)
			assert.Equal(t, id, ref.ID())
			assert.Equal(t, "電子機器", ref.Name())
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("id がゼロ値の場合、ErrInvalidCategoryID を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewCategoryRef(uuid.UUID{}, "電子機器")
			require.ErrorIs(t, err, ErrInvalidCategoryID)
			assert.Equal(t, CategoryRef{}, ref)
		})

		t.Run("名称が空の場合、ErrInvalidCategoryName を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewCategoryRef(id, "")
			require.ErrorIs(t, err, ErrInvalidCategoryName)
			assert.Equal(t, CategoryRef{}, ref)
		})

		t.Run("名称が最大長を超える場合、ErrInvalidCategoryName を返す", func(t *testing.T) {
			t.Parallel()
			ref, err := NewCategoryRef(id, strings.Repeat("あ", maxRefNameLength+1))
			require.ErrorIs(t, err, ErrInvalidCategoryName)
			assert.Equal(t, CategoryRef{}, ref)
		})
	})
}

func TestCategoryRef_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時のカテゴリ ID を返す", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "category_ref_id")
			ref, err := NewCategoryRef(id, "電子機器")
			require.NoError(t, err)

			assert.Equal(t, id, ref.ID())
		})
	})
}

func TestCategoryRef_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時のカテゴリ名称を返す", func(t *testing.T) {
			t.Parallel()

			ref, err := NewCategoryRef(uuidtestkit.NewTestFromSalt(t, "category_ref_id"), "電子機器")
			require.NoError(t, err)

			assert.Equal(t, "電子機器", ref.Name())
		})
	})
}

func TestStatusRef_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時のステータス ID を返す", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "status_ref_id")
			ref, err := NewStatusRef(id, "在庫あり")
			require.NoError(t, err)

			assert.Equal(t, id, ref.ID())
		})
	})
}

func TestStatusRef_Name(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時のステータス名称を返す", func(t *testing.T) {
			t.Parallel()

			ref, err := NewStatusRef(uuidtestkit.NewTestFromSalt(t, "status_ref_id"), "在庫あり")
			require.NoError(t, err)

			assert.Equal(t, "在庫あり", ref.Name())
		})
	})
}
