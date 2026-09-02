//go:generate mockgen -source=$GOFILE -destination=mock/mock_$GOFILE.gen.go -package=mock_$GOPACKAGE

// Package query は、商品の読み取り専用クエリ（QueryService）のインターフェースを提供します。
// 画像パスの参照有無は商品を経由せず product_images を横断して引く読みで、分解すると必要としない
// 商品を全件 materialize することになるため QueryService として定義します
// （docs/design/data-access-pattern.md の §3.3）。
package query

import "context"

// ProductImageQueryService は、商品画像の横断的な参照照合を提供する QueryService です。
type ProductImageQueryService interface {
	// FilterExistingImagePaths は、paths のうち、いずれかの商品が現在の画像として参照しているものを
	// 返します。重複は取り除き、順序は保証しません。paths が空の場合は空を返します。
	//
	// 返らなかったパスは、どの商品からも参照されていないことを意味します。置き換えで論理削除された
	// 画像は現在の参照ではないため、参照元として数えません。
	FilterExistingImagePaths(ctx context.Context, paths []string) ([]string, error)
}
