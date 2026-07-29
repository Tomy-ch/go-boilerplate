package product

const (
	minNameLength = 1
	maxNameLength = 255
	minQuantity   = 0
	minThreshold  = 0
	// minRefNameLength / maxRefNameLength は、ステータス／カテゴリ参照が保持する名称の長さ範囲です。
	// 参照元マスタ（商品ステータスマスタ／商品カテゴリマスタ）の名称長制約に揃えています。
	minRefNameLength = 1
	maxRefNameLength = 100
	// initialVersion は、生成直後の商品が持つ楽観ロックのバージョンです。更新のたびに 1 つ進みます。
	initialVersion = 1
)
