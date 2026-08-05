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
	// maxThreshold は、在庫警告閾値の上限です。閾値は在庫数と比較されるため、在庫数の上限を
	// 超える閾値は「常に警告」を意味し閾値として機能しません。在庫数と同じ幅で表現します。
	maxThreshold = math.MaxInt32
	// minRefNameLength / maxRefNameLength は、ステータス／カテゴリ参照が保持する名称の長さ範囲です。
	// 参照元マスタ（商品ステータスマスタ／商品カテゴリマスタ）の名称長制約に揃えています。
	minRefNameLength = 1
	maxRefNameLength = 100
	// initialVersion は、生成直後の商品が持つ楽観ロックのバージョンです。更新のたびに 1 つ進みます。
	initialVersion = 1
)
