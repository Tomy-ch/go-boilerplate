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
	// minRefNameLength / maxRefNameLength は、ステータス／カテゴリ参照が保持する名称の長さ範囲です。
	// 参照元マスタ（商品ステータスマスタ／商品カテゴリマスタ）の名称長制約に揃えています。
	minRefNameLength = 1
	maxRefNameLength = 100
	// initialVersion は、生成直後の商品が持つ楽観ロックのバージョンです。更新のたびに 1 つ進みます。
	initialVersion = 1
)
