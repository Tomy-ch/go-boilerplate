package product

import "math"

const (
	minNameLength = 1
	maxNameLength = 255
	minQuantity   = 0
	// maxQuantity は、在庫数の上限です。在庫は 32bit 整数幅で表現するため、
	// 増減の結果がこれを超える在庫は保持できません。
	maxQuantity  = math.MaxInt32
	minThreshold = 0
	// maxThreshold は、在庫警告閾値の上限です。閾値は在庫数と突き合わせる値であり、在庫数と同じ
	// 32bit 整数幅で表現します。
	maxThreshold = math.MaxInt32
	// maxImages は、1 商品が保持できる画像の枚数の上限です。
	//
	// placeholder 定数。画像は商品の読み出しごとに集約へ materialize され、一覧では
	// 商品数 × 枚数だけ行が載るため、集約を小さく保つ目的で上限を置きます
	// （internal/domain/README.md の Aggregate Boundary）。20 は商品ギャラリーとして十分広く、
	// 100 件の一覧でも画像行を 2000 行に収める値として選んでいます。
	// 枚数の業務要件が立てば docs/spec/product/domain.md で改めます。
	maxImages = 20
	// minImageDisplaySort は、商品画像の表示順の下限です。表示順は 1 から数えます。
	minImageDisplaySort = 1
	// maxImageDisplaySort は、商品画像の表示順の上限です。表示順は 16bit 整数幅で表現します。
	maxImageDisplaySort = math.MaxInt16
	// minRefNameLength / maxRefNameLength は、ステータス／カテゴリ参照が保持する名称の長さ範囲です。
	// 参照元マスタ（商品ステータスマスタ／商品カテゴリマスタ）の名称長制約に揃えています。
	minRefNameLength = 1
	maxRefNameLength = 100
	// initialVersion は、生成直後の商品が持つ楽観ロックのバージョンです。
	initialVersion = 1
)
