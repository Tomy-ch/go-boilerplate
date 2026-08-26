package product

import (
	"testing"

	"go-boilerplate/internal/apperror"
	"go-boilerplate/pkg/uuid"
	uuidtestkit "go-boilerplate/pkg/uuid/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImage(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("渡した属性をそのまま保持する", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "product_image_id")

			img := NewImage(id, ImageAttributes{ImagePath: "products/earphone.png", DisplaySort: 3})

			assert.Equal(t, id, img.ID())
			assert.Equal(t, "products/earphone.png", img.ImagePath())
			assert.Equal(t, 3, img.DisplaySort())
		})

		t.Run("不変条件は集約が検証するため、単体では検証せずに構築できる", func(t *testing.T) {
			t.Parallel()

			img := NewImage(uuid.UUID{}, ImageAttributes{ImagePath: "", DisplaySort: 0})

			assert.True(t, img.ID().IsNil())
			assert.Empty(t, img.ImagePath())
			assert.Equal(t, 0, img.DisplaySort())
		})
	})
}

func TestImage_ID(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時のIDを返す", func(t *testing.T) {
			t.Parallel()

			id := uuidtestkit.NewTestFromSalt(t, "product_image_id")

			assert.Equal(t, id, NewImage(id, ImageAttributes{ImagePath: "products/a.png", DisplaySort: 1}).ID())
		})
	})
}

func TestImage_ImagePath(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時の画像パスを返す", func(t *testing.T) {
			t.Parallel()

			img := mustImage(t, "product_image_id", "products/earphone.png", 1)

			assert.Equal(t, "products/earphone.png", img.ImagePath())
		})
	})
}

func TestImage_DisplaySort(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("構築時の表示順を返す", func(t *testing.T) {
			t.Parallel()

			img := mustImage(t, "product_image_id", "products/earphone.png", 7)

			assert.Equal(t, 7, img.DisplaySort())
		})
	})
}

func Test_validateImages(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("表示順が重複せず範囲内の画像は検証を通る", func(t *testing.T) {
			t.Parallel()

			images := []Image{
				mustImage(t, "product_image_id_1", "products/a.png", minImageDisplaySort),
				mustImage(t, "product_image_id_2", "products/b.png", maxImageDisplaySort),
			}

			require.NoError(t, validateImages(images))
		})

		t.Run("画像を持たない場合は検証を通る", func(t *testing.T) {
			t.Parallel()

			require.NoError(t, validateImages(nil))
		})
	})

	t.Run("異常系", func(t *testing.T) {
		t.Parallel()

		t.Run("IDが未設定の場合、ErrInvalidIDを返す", func(t *testing.T) {
			t.Parallel()

			images := []Image{NewImage(uuid.UUID{}, ImageAttributes{ImagePath: "products/a.png", DisplaySort: 1})}

			err := validateImages(images)

			require.ErrorIs(t, err, ErrInvalidID)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("画像パスが空の場合、ErrInvalidImagePathを返す", func(t *testing.T) {
			t.Parallel()

			images := []Image{mustImage(t, "product_image_id_1", "", 1)}

			err := validateImages(images)

			require.ErrorIs(t, err, ErrInvalidImagePath)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("表示順が下限未満の場合、ErrInvalidImageDisplaySortを返す", func(t *testing.T) {
			t.Parallel()

			images := []Image{mustImage(t, "product_image_id_1", "products/a.png", minImageDisplaySort-1)}

			err := validateImages(images)

			require.ErrorIs(t, err, ErrInvalidImageDisplaySort)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("表示順が上限超過の場合、ErrInvalidImageDisplaySortを返す", func(t *testing.T) {
			t.Parallel()

			images := []Image{mustImage(t, "product_image_id_1", "products/a.png", maxImageDisplaySort+1)}

			err := validateImages(images)

			require.ErrorIs(t, err, ErrInvalidImageDisplaySort)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})

		t.Run("同一商品内で表示順が重複する場合、ErrDuplicateImageDisplaySortを返す", func(t *testing.T) {
			t.Parallel()

			images := []Image{
				mustImage(t, "product_image_id_1", "products/a.png", 1),
				mustImage(t, "product_image_id_2", "products/b.png", 1),
			}

			err := validateImages(images)

			require.ErrorIs(t, err, ErrDuplicateImageDisplaySort)
			require.ErrorIs(t, err, apperror.ErrValidation)
		})
	})
}

func Test_sortImagesByDisplaySort(t *testing.T) {
	t.Parallel()

	t.Run("正常系", func(t *testing.T) {
		t.Parallel()

		t.Run("表示順の昇順に並べ替えた新しいスライスを返す", func(t *testing.T) {
			t.Parallel()

			images := []Image{
				mustImage(t, "product_image_id_3", "products/c.png", 3),
				mustImage(t, "product_image_id_1", "products/a.png", 1),
				mustImage(t, "product_image_id_2", "products/b.png", 2),
			}

			sorted := sortImagesByDisplaySort(images)

			require.Len(t, sorted, 3)
			assert.Equal(t, 1, sorted[0].DisplaySort())
			assert.Equal(t, 2, sorted[1].DisplaySort())
			assert.Equal(t, 3, sorted[2].DisplaySort())
			// 元のスライスは並べ替えの影響を受けない。
			assert.Equal(t, 3, images[0].DisplaySort())
		})
	})
}
