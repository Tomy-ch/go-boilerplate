package product

import (
	"fmt"
	"slices"

	"go-boilerplate/pkg/uuid"
	"go-boilerplate/pkg/xerrors"
)

// Image は、商品画像を表す商品集約の子エンティティです。id は更新をまたいで同じ画像を指し続けるため、
// 内容の差し替えは新しい id の画像として表現します。imagePath はオブジェクトストレージ上のキーで、
// 表示 URL は上位が配信ベース URL と組み立てます。displaySort は同一商品内での表示順であり、表示順の
// 入れ替えで使い回されるため同一性を担いません。
type Image struct {
	id          uuid.UUID
	imagePath   string
	displaySort int
}

// ImageAttributes は、商品画像の構築に必要な属性一式です。
type ImageAttributes struct {
	ImagePath   string
	DisplaySort int
}

// NewImage は、商品画像を構築します。不変条件は集約が保持する集合として検証されるため
// （兄弟間の表示順の重複はここでは判定できません）、ここでは組み立てのみを行います。
func NewImage(id uuid.UUID, attrs ImageAttributes) Image {
	return Image{
		id:          id,
		imagePath:   attrs.ImagePath,
		displaySort: attrs.DisplaySort,
	}
}

// ID は、商品画像の ID を返します。
func (i Image) ID() uuid.UUID { return i.id }

// ImagePath は、画像パスを返します。
func (i Image) ImagePath() string { return i.imagePath }

// DisplaySort は、同一商品内での表示順を返します。
func (i Image) DisplaySort() int { return i.displaySort }

// validateImages は、商品が保持する画像の集合が満たすべき不変条件を検証します。
// ID が未設定、画像パスが空、表示順が保持できる範囲外の場合はそれぞれ ErrInvalidID /
// ErrInvalidImagePath / ErrInvalidImageDisplaySort を、同一商品内で表示順が重複する場合は
// ErrDuplicateImageDisplaySort を返します。画像を持たない商品は許容します。
func validateImages(images []Image) error {
	seen := make(map[int]struct{}, len(images))
	for _, img := range images {
		if img.id.IsNil() {
			return xerrors.Wrap(ErrInvalidID, "image id is required")
		}
		if img.imagePath == "" {
			return xerrors.Wrap(ErrInvalidImagePath, "image path is required")
		}
		if img.displaySort < minImageDisplaySort || img.displaySort > maxImageDisplaySort {
			return xerrors.Wrap(
				ErrInvalidImageDisplaySort,
				fmt.Sprintf("image displaySort must be between %d and %d, got %d", minImageDisplaySort, maxImageDisplaySort, img.displaySort),
			)
		}
		if _, ok := seen[img.displaySort]; ok {
			return xerrors.Wrap(ErrDuplicateImageDisplaySort, fmt.Sprintf("image displaySort %d is duplicated", img.displaySort))
		}
		seen[img.displaySort] = struct{}{}
	}
	return nil
}

// sortImagesByDisplaySort は、画像を表示順の昇順に並べた新しいスライスを返します。
// 集約の構築時に一度だけ通すことで、どの読み出し経路から取り出しても並びが同じになります。
func sortImagesByDisplaySort(images []Image) []Image {
	sorted := make([]Image, len(images))
	copy(sorted, images)
	slices.SortFunc(sorted, func(a, b Image) int { return a.displaySort - b.displaySort })

	return sorted
}
